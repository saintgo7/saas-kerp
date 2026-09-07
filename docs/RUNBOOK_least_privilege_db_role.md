# 런북 — 애플리케이션 DB 계정을 `kerp` 에서 `kerp_app` 으로 전환

상태: **미실행 (준비만 완료)**
소유: 인프라 + Go + DB 공동
관련 파일: `deployments/docker/docker-compose.yml`(api·worker 서비스), `db/migrations/000016_least_privilege_app_role.up.sql`, `db/migrations/000017_rls_hardening.up.sql`

---

## 왜 지금 바꾸면 안 되는가

현재 API 와 워커는 `kerp` 계정으로 접속합니다. `kerp` 는 스키마 소유자이므로
**Row Level Security 를 통째로 우회**합니다. 감사에서 P0 로 지적된 그대로입니다.

그런데 이 계정을 비소유자인 `kerp_app` 으로 그냥 바꾸면 RLS 가 그 순간부터 실제로
적용되기 시작합니다. RLS 정책은 `app.current_tenant` GUC 를 읽는데, **Go 쪽에서 요청마다
이 GUC 를 설정하는 코드가 아직 배포되지 않았습니다.** 설정되지 않은 GUC 와 비교하면
정책이 어떤 행도 통과시키지 않습니다.

결과는 "에러"가 아니라 **조용한 빈 응답**입니다.

- 컨테이너는 정상 기동하고 `/health` 도 200 을 반환합니다.
- 로그인도 되고 화면도 뜹니다.
- 그런데 전표·계정·거래처 조회가 전부 0건으로 나옵니다.
- 모니터링·헬스체크·알림 중 어느 것도 이 상태를 장애로 인식하지 못합니다.

즉 **전환 순서를 어기면 서비스가 죽습니다.** 그것도 죽은 줄 모르는 형태로 죽습니다.
데이터가 사라진 것처럼 보이기 때문에 운영 중이라면 즉시 사용자 신고가 들어옵니다.

---

## 전환 순서 (이 순서를 바꾸지 마십시오)

### 1단계 — Go: 요청마다 `app.current_tenant` 설정 (Go 담당)

- 테넌트 미들웨어가 확보한 `company_id` 를 커넥션 단위로 `SET LOCAL app.current_tenant`
  하도록 구현하고, 트랜잭션/커넥션 풀 재사용 경로에서 값이 새지 않는지 확인합니다.
- **아직 DSN 은 건드리지 않습니다.** 이 단계에서는 `kerp` 로 접속하므로 GUC 를 설정해도
  동작이 달라지지 않습니다 — 즉 무해하게 선반영할 수 있습니다.

### 2단계 — 스테이징에서 GUC 코드 검증

`kerp` 접속 상태 그대로 스테이징에 배포한 뒤 확인합니다.

```sql
-- 임의의 요청 처리 중 실제 세션에서
SELECT current_setting('app.current_tenant', true);
```

- 값이 비어 있지 않아야 합니다.
- 여러 테넌트로 번갈아 요청했을 때 값이 따라 바뀌어야 합니다.
- 커넥션 풀에서 재사용된 커넥션에 이전 테넌트 값이 남지 않아야 합니다.

### 3단계 — 마이그레이션으로 `kerp_app` 역할 준비 (DB 담당)

`000016_least_privilege_app_role` 이 역할을 만듭니다. 운영에 아직 적용되지 않았다면
먼저 적용합니다. 비밀번호는 반드시 새로 생성해 GitHub Secrets 에 `STAGING_APP_DB_PASSWORD`
/ `PRODUCTION_APP_DB_PASSWORD` 로 등록합니다.

```bash
openssl rand -base64 32
```

### 4단계 — 스테이징에서 DSN 전환

`deployments/docker/docker-compose.yml` 의 api·worker 서비스에서 주석 처리된 두 줄을
살리고 기존 두 줄을 주석 처리합니다.

```yaml
- KERP_DATABASE_USER=kerp_app
- KERP_DATABASE_PASSWORD=${APP_DB_PASSWORD:?APP_DB_PASSWORD must be set}
```

CD 의 `.env` 생성 스텝(`.github/workflows/cd.yml`)에도 `APP_DB_PASSWORD` 를 추가해야
합니다.

**검증 (이걸 통과하기 전에는 운영으로 넘어가지 마십시오):**

- 테넌트 A 로 로그인해 전표 목록이 **0건이 아닌지** 확인합니다. 0건이면 GUC 가 설정되지
  않은 것이므로 즉시 3단계 이전으로 되돌립니다.
- 테넌트 A 의 토큰으로 테넌트 B 의 리소스 ID 를 직접 조회했을 때 404/403 이 나오는지
  확인합니다.
- 마이그레이션은 여전히 `kerp` 로 실행해야 합니다. `kerp_app` 에는 DDL 권한이 없습니다.

### 5단계 — 운영 전환

스테이징에서 위 검증이 모두 통과한 뒤에만 같은 변경을 운영에 적용합니다.

---

## 롤백

증상(전 조회 0건)이 보이면 즉시 DSN 을 `kerp` 로 되돌리고 재기동합니다.
데이터는 삭제되지 않았습니다 — RLS 가 가린 것뿐이므로 계정을 되돌리면 그대로 보입니다.

```bash
# 운영 호스트에서
cd /home/blackpc/saas-kerp-production/deployments/docker
# docker-compose.yml 의 KERP_DATABASE_USER 를 kerp 로 되돌린 뒤
docker compose -p kerp-production up -d api worker
```

---

## 함께 정리할 것

- `postgres-exporter` 도 아직 `kerp` 로 접속합니다(`docker-compose.yml`). 읽기 전용
  모니터링 역할이 생기면 그쪽으로 옮깁니다. 현재는 `docker inspect` 로 DSN 안의
  비밀번호가 그대로 보입니다.
- Grafana PostgreSQL 데이터소스도 동일합니다
  (`monitoring/grafana/provisioning/datasources/datasources.yml`).
