# Чистая JavaScript Архитектура

## 🎯 Концепция

GPT теперь может возвращать **ТОЛЬКО JavaScript код**. Никакого обычного текста!

Любой ответ GPT автоматически выполняется как JavaScript в песочнице с доступом к Teamwork API.

## 🔄 Поток выполнения

```
Пользователь → GPT → JavaScript код → Выполнение → message() → Пользователь
                ↑                                      ↓
                ← output() ← Продолжение ← GPT ←────────┘
```

## 🚨 Ключевые принципы

### ❌ Что НЕЛЬЗЯ:
- Возвращать обычный текст
- Использовать `return` в глобальном контексте
- Отвечать без `message()`

### ✅ Что НУЖНО:
- Возвращать только JavaScript код
- Использовать `message()` для ВСЕХ ответов пользователю
- Использовать `output()` для передачи данных себе

## 📤 Функция message()

**Единственный способ ответить пользователю!**

```javascript
message("Привет! 👋");
message("Вот ваши проекты:");
message("1. Проект А");
message("2. Проект Б");
```

- Каждый вызов = отдельное сообщение пользователю
- Можно вызывать многократно
- Поддерживает эмодзи и форматирование

## 📥 Функция output()

**Передача данных обратно GPT для продолжения работы**

```javascript
let data = teamwork.listProjects();
let stats = {total: data.length, active: data.filter(p => p.status === 'active').length};
output(stats); // GPT получит эти данные в следующем запросе
```

- После `output()` GPT получает новый контекст с данными
- Используется для многоступенчатой обработки
- Можно комбинировать с `message()`

## 🔧 Доступные API

### Чтение данных (немедленно):
- `teamwork.listProjects(status?)`
- `teamwork.listTasks(params?)`
- `teamwork.getCurrentProject()`

### Модификация данных (с подтверждением):
- `teamwork.createProject(name, description?)`
- `teamwork.updateProject(id, params)`
- `teamwork.deleteProject(id)`
- `teamwork.createTask(title, params?)`
- `teamwork.updateTask(id, params)`
- `teamwork.deleteTask(id)`
- `teamwork.setCurrentProject(id)`

### Утилиты:
- `message(text)` - отправка сообщений
- `output(data)` - возврат данных
- `debug(obj)` - отладка
- `fetch(url, options)` - HTTP запросы
- `setTimeout()`, `JSON.*`, `Math.*`, `Date.*`

## 📋 Примеры использования

### Простой ответ:
```javascript
message("👋 Привет! Как дела?");
```

### Показать данные:
```javascript
let projects = teamwork.listProjects();
message("📊 У вас " + projects.length + " проектов:");
projects.forEach(p => message("• " + p.title + " (" + p.status + ")"));
```

### Создание с подтверждением:
```javascript
message("🚀 Создаю проект...");
let result = teamwork.createProject("Новый проект", "Описание");
message("📋 " + result.description);
```

### Многоступенчатый анализ:
```javascript
message("🔍 Анализирую ваши данные...");
let projects = teamwork.listProjects();
let tasks = teamwork.listTasks();

let analysis = {
  projects: projects.length,
  tasks: tasks.length,
  ratio: tasks.length / projects.length
};

message("📈 Найдено: " + analysis.projects + " проектов, " + analysis.tasks + " задач");
output(analysis); // Передаю данные для дальнейшего анализа
```

### API запросы:
```javascript
message("🌐 Получаю данные из внешнего API...");
fetch('https://api.github.com/users/octocat')
  .then(r => r.json())
  .then(user => {
    message("✅ Пользователь: " + user.name);
    message("📊 Репозиториев: " + user.public_repos);
  })
  .catch(err => message("❌ Ошибка: " + err.message));
```

## 🎯 Преимущества новой архитектуры

1. **Простота**: Один способ взаимодействия - JavaScript
2. **Гибкость**: GPT может делать сложные вычисления и API запросы
3. **Интерактивность**: Множественные `message()` для живого общения
4. **Продолжения**: `output()` для многоступенчатых процессов
5. **Чистота**: Нет путаницы между function calls и обычными ответами

## 🚀 Результат

Теперь GPT - это **JavaScript-программист**, который:
- Пишет код для решения задач пользователя
- Общается через `message()`
- Может делать сложные вычисления и API запросы
- Поддерживает многоступенчатые процессы через `output()`
- Работает с Teamwork API для управления проектами

**Единственное правило: Всегда используй `message()` для ответа пользователю!** 🎯 