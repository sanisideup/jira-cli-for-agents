# RFC: Local Project Templates Directory

**Created:** 2026-02-06
**Status:** Implementing
**Branch:** feature/local-templates

## Problem Statement

JCFA templates currently live only in `~/.jcfa/templates/`. This creates issues:

1. **No version control** - Templates aren't tracked with project code
2. **No portability** - Team members must manually copy templates
3. **No project-specific overrides** - Can't have different templates per project
4. **Setup friction** - New users must run `jcfa template init` + copy custom templates

## Solution

Add support for project-local templates directory with fallback to user templates.

### Template Resolution Order

1. **Explicit**: `--templates-dir` flag (highest priority)
2. **Local**: `./.jcfa/templates/` (project-local)
3. **Config**: `templates_dir` in `~/.jcfa/config.yaml`
4. **User**: `~/.jcfa/templates/`
5. **Built-in**: Embedded defaults (lowest priority)

## Implementation Phases

### Phase 1: Config Changes
- Add `TemplatesDir` field to `Config` struct in `pkg/config/config.go`

### Phase 2: Template Resolver
- Create `pkg/template/resolver.go` with `TemplateResolver` struct
- Implement `Resolve(name string)` and `List()` methods
- Add `TemplateInfo` struct with Name, Path, Source fields

### Phase 3: Update Create Command
- Add `--templates-dir` flag to `cmd/create.go`
- Use resolver for template loading

### Phase 4: Update Template List Command
- Show source labels (local/config/user/builtin) in output

### Phase 5: Update Template Show Command
- Display resolved path in output

### Phase 6: Add Template Init --local
- Add `--local` flag to initialize templates in `.jcfa/templates/`

## Migration Path

No breaking changes. Existing workflows continue to work unchanged.

## Future Enhancements (Out of Scope)

- [ ] Environment variable `JCFA_TEMPLATES_DIR` override
- [ ] Template inheritance (extend base templates)
- [ ] Remote template repositories
- [ ] Template validation command
