# Windows 방화벽 포트 설정

> HTTPS 인증서 발급을 위해 포트 80, 443 오픈 필요

---

## Windows PowerShell (관리자 권한)에서 실행

```powershell
# 포트 80 (HTTP - Let's Encrypt 검증용)
New-NetFirewallRule -DisplayName "HTTP 80" -Direction Inbound -Protocol TCP -LocalPort 80 -Action Allow

# 포트 443 (HTTPS)
New-NetFirewallRule -DisplayName "HTTPS 443" -Direction Inbound -Protocol TCP -LocalPort 443 -Action Allow

# 확인
Get-NetFirewallRule -DisplayName "HTTP 80" | Format-List
Get-NetFirewallRule -DisplayName "HTTPS 443" | Format-List
```

---

## 또는 GUI로 설정

1. Windows + R > `wf.msc`
2. **Inbound Rules** > **New Rule**
3. **Port** 선택 > Next
4. **TCP** > **Specific local ports**: `80, 443`
5. **Allow the connection**
6. 모든 프로필 체크 (Domain, Private, Public)
7. Name: `K-ERP HTTP/HTTPS`

---

## 설정 후 테스트

```bash
# Mac에서 포트 확인
nc -zv 61.245.248.246 80
nc -zv 61.245.248.246 443
```

---

## 방화벽 설정 완료 후

WSL에서 인증서 발급 재시도:
```bash
sudo certbot certonly --standalone -d erp.abada.kr --agree-tos -m admin@abada.kr --non-interactive
```

---

*2026-01-16*
