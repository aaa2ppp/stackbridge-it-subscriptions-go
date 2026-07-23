# Сервис для агрегации данных об онлайн подписках пользователей (GO)

[Техническое задание](./docs/TS.md)

## Быстрый старт

### Клонирование репозитория

```bash
git clone https://github.com/aaa2ppp/stackbridge-it-subscriptions-go.git
cd stackbridge-it-subscriptions-go
```

### Переменные окружения

Пример всех используемых переменных окружения находится в файле [`dot.env.example`](./dot.env.example).  
Скопируйте его и отредактируйте под свои нужды:

```bash
cp dot.env.example .env

# Сгенерировать случайный пароль для базы данных
echo -e "\nDB_PASSWORD='$(head -c16 /dev/urandom | base64)'" >> .env
```

### Локальный запуск

```sh
# Загрузить переменные окружения
. dev-env

# Установить зависимости и проверить утилиты (при необходимости будут установлены)
make deps check-tools

# Запустить приложение
make run
```

### Запуск в Docker

```sh
# Запустить приложение в контейнере
make docker-run
```

### Swagger

По умолчанию Swagger доступен на [http://localhost:8080/swagger/](http://localhost:8080/swagger/)

### Уборка

```sh
# Удалить контейнеры и том базы данных
make docker-down-volumes

# Удалит локально собранные бинарники и временные файлы
make clean
```

### Управление через Makefile

Все доступные команды `make` можно посмотреть, выполнив:

```bash
make help
```

## Ограничения, особенности и отступления от ТЗ
