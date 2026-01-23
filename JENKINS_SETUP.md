# Jenkins Setup Guide

## Запуск Jenkins через Docker

### 1. Запустить Jenkins контейнер

```bash
docker-compose up -d
```

Это создаст Jenkins контейнер, который будет доступен по адресу `http://localhost:8080`

### 2. Получить начальный пароль

```bash
docker exec jenkins cat /var/jenkins_home/secrets/initialAdminPassword
```

Скопируйте пароль и используйте его при первом входе в Jenkins.

### 3. Настроить Jenkins

1. Откройте `http://localhost:8080` в браузере
2. Введите начальный пароль
3. Установите рекомендуемые плагины (включая Docker Pipeline plugin)
4. Создайте административного пользователя

### 4. Настроить Docker в Jenkins

Jenkins контейнер уже настроен для использования Docker через volume mount (`/var/run/docker.sock`).

### 5. Создать Pipeline Job

1. В Jenkins Dashboard нажмите "New Item"
2. Выберите "Pipeline"
3. Назовите job (например, "linux-pod-ci")
4. Нажмите "OK"
5. В разделе "Pipeline":
   - **Definition**: Измените с "Pipeline script" на **"Pipeline script from SCM"** (важно!)
   - **SCM**: Выберите **"Git"**
   - **Repository URL**: Введите URL вашего репозитория (например, `https://github.com/your-username/your-repo.git`)
   - **Credentials**: Если репозиторий приватный, добавьте credentials
   - **Branches to build**: 
     - Нажмите "Add" рядом с "Branch Specifier"
     - Введите `*/main` (если основная ветка называется `main`)
     - Или `*/master` (если основная ветка называется `master`)
     - **Важно**: Убедитесь, что указана правильная ветка, которая существует в репозитории!
   - **Script Path**: Убедитесь, что указано `Jenkinsfile`
6. Нажмите "Save"

### 6. Запустить Pipeline

Нажмите "Build Now" для запуска pipeline.

## Pipeline Stages

Pipeline выполняет следующие этапы:

1. **Checkout** - Клонирует репозиторий
2. **Setup Go** - Устанавливает или обновляет Go 1.22
3. **Format Check** - Проверяет форматирование кода (`make fmt-check`)
4. **Lint** - Запускает линтер (`make lint`)
5. **Unit Tests** - Запускает unit тесты (`make test-ci`)

## Остановка Jenkins

```bash
docker-compose down
```

Для остановки с удалением данных:

```bash
docker-compose down -v
```

## Доступ к Jenkins

- URL: http://localhost:8080
- Порт 50000 используется для Jenkins agents (если нужно)
