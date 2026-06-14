<!-- graphify 개념-그래프 분해 시도 결과를 기록하는 스윕 산출물 -->
# Graphify Decomposition Status

- Tool: graphify 0.8.39, backend `pamout` (사용자 LLM 서버, SSH 터널 경유).
- Date: 2026-06-14.

## 결과: 부분 실패 (그래프 미생성)

- AST 추출 단계는 성공: 367개 코드 파일, 144 docs, 16 images 스캔 완료. AST 캐시는 `cache/`에 저장됨(gitignore 처리).
- 시맨틱(LLM) 추출 단계 실패: 10/10 청크 모두 `Connection error`로 실패. 백엔드 연결 프로브 결과 HTTP 000 (도달 불가).
- 따라서 `graph.json` / `GRAPH_REPORT.md` 미생성. `cluster-only` 및 `diagnose multigraph`도 그래프가 없어 실행 불가.

## 영향

- 그래프 기반 커뮤니티/허브/고아 노드 분석은 이번 스윕에서 수행 불가.
- 대신 직접 코드 리뷰 + 빌드/테스트 실행으로 문제를 식별했다(상호참조 grep으로 미배선 핸들러·고아 서비스 탐지).

## 재시도 방법

```
source /Users/saint/01_DEV/_project-sweep-2026-06-13/.graphify-backend.env
graphify extract . --backend pamout    # AST 캐시는 재사용됨
graphify cluster-only . --backend pamout
graphify diagnose multigraph --graph graphify-out/graph.json
```

백엔드(터널)가 복구되면 AST 캐시가 살아있어 시맨틱 단계만 다시 돌면 된다.
