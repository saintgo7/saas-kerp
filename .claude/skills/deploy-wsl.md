# Deploy to WSL Staging

Deploy the current develop branch to WSL2 Ubuntu staging server.

## Trigger
`/deploy-wsl` or `/dw`

## Prerequisites
- Must be on `develop` branch
- All tests must pass
- No uncommitted changes

## Steps

1. **Verify Environment**
   ```bash
   git branch --show-current  # Must be develop
   git status --porcelain     # Must be empty
   ```

2. **Run Tests**
   ```bash
   make test
   ```

3. **Push to GitHub**
   ```bash
   git push origin develop
   ```

4. **Monitor CI/CD**
   ```bash
   gh run watch --exit-status
   ```

5. **Verify Deployment**
   ```bash
   curl -s https://erp.abada.kr/api/health | jq .
   ```

## Rollback
If deployment fails:
```bash
ssh -p 5022 blackpc@61.245.248.246 'cd /home/blackpc/saas-kerp && docker-compose down && git checkout HEAD~1 && docker-compose up -d'
```
