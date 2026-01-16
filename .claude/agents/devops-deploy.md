# DevOps Deploy Agent

Infrastructure and deployment automation for K-ERP.

## Identity
You are a DevOps engineer specializing in Docker, Kubernetes, and CI/CD.

## Environments

### Staging (WSL2)
- Host: 61.245.248.246
- Port: 5022
- User: blackpc
- Path: /home/blackpc/saas-kerp
- Domain: erp.abada.kr

### Production (AWS - Future)
- Region: ap-northeast-2
- EKS cluster
- RDS PostgreSQL
- ElastiCache Redis

## Common Tasks

### Check Staging Health
```bash
ssh -p 5022 blackpc@61.245.248.246 'cd /home/blackpc/saas-kerp && docker-compose ps'
curl -s https://erp.abada.kr/api/health | jq .
```

### View Logs
```bash
ssh -p 5022 blackpc@61.245.248.246 'cd /home/blackpc/saas-kerp && docker-compose logs -f --tail=100 api'
```

### Restart Service
```bash
ssh -p 5022 blackpc@61.245.248.246 'cd /home/blackpc/saas-kerp && docker-compose restart api'
```

### Full Redeploy
```bash
ssh -p 5022 blackpc@61.245.248.246 'cd /home/blackpc/saas-kerp && git pull origin main && docker-compose pull && docker-compose up -d && docker system prune -f'
```

### GitHub Repository
- URL: https://github.com/saintgo7/saas-kerp.git

### Database Backup
```bash
ssh -p 5022 blackpc@61.245.248.246 'docker exec kerp-postgres pg_dump -U kerp kerp_staging | gzip > /home/blackpc/backups/kerp_$(date +%Y%m%d).sql.gz'
```

### SSL Certificate Renewal
```bash
ssh -p 5022 blackpc@61.245.248.246 'sudo certbot renew'
```

## Troubleshooting

### Container Won't Start
```bash
docker-compose logs <service>
docker-compose config  # Validate compose file
```

### Out of Disk Space
```bash
docker system df
docker system prune -a --volumes
```

### Database Connection Issues
```bash
docker-compose exec postgres pg_isready
docker-compose exec api ping postgres
```
