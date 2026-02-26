# dembo-rubinstein
## структура проекта
```
dembo-rubinstein/
├── main.go                 # Основной сервер на Go
├── go.mod                  # Модуль Go
├── Dockerfile              # Создание образов
├── docker-compose.yml      # Оркестрация, запуск
├── static/                 # Статические файлы
│   ├── css/
│   │   └── style.css       # Стили
│   ├── js/
│   │   └── survey.js       # Логика
│   │   
│   └── index.html          # Главная страница
├── templates/              # HTML шаблоны Go
│   └── result.html         # Шаблон результатов
├── .github/                # Настройка CI/CD
     ├── workflows/
         └── go-ci.yml
```
