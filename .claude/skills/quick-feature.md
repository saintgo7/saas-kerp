# Quick Feature Development

Rapidly develop a new feature with proper structure.

## Trigger
`/qf <module> <feature-name>` or `/quick-feature`

## Arguments
- `module`: backend, frontend, worker
- `feature-name`: kebab-case feature name

## Process

### Backend Feature
1. Create feature branch
   ```bash
   git checkout -b feature/${feature-name}
   ```

2. Generate handler
   ```
   internal/handler/${feature}_handler.go
   ```

3. Generate service
   ```
   internal/service/${feature}_service.go
   ```

4. Generate repository
   ```
   internal/repository/${feature}_repository.go
   ```

5. Generate tests
   ```
   internal/handler/${feature}_handler_test.go
   internal/service/${feature}_service_test.go
   ```

6. Update routes
   ```
   internal/handler/router.go
   ```

### Frontend Feature
1. Create component
   ```
   web/src/components/${Feature}/index.tsx
   web/src/components/${Feature}/${Feature}.tsx
   web/src/components/${Feature}/${Feature}.test.tsx
   ```

2. Create API hook
   ```
   web/src/hooks/use${Feature}.ts
   ```

3. Update types
   ```
   web/src/types/${feature}.ts
   ```

## Completion
```bash
make test
git add -A
git commit -m "feat(${module}): Add ${feature-name}"
```
