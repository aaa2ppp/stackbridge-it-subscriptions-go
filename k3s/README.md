
## Основные сценарии

### CREATE (первое развёртывание)

```bash
make ns-create
make migrate-build migrate-install
make server-build server-install
make db-create
make migrate-run
make server-create
make server-start
```

### RELOAD (только код сервера, без миграций)

```bash
make server-reload
```

### RELOAD (миграции + код сервера)

```bash
make migrate-build migrate-install   # если мигратор менялся
make server-build server-install     # если сервер менялся
make server-stop                     # останавливаем поды
make migrate-run                     # применяем миграции
make server-start                    # запускаем поды с новым кодом
```

### DESTROY & CLEAR ALL

```bash
make ns-destroy
make clean
```
