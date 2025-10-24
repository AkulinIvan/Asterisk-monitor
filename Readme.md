# Asterisk Monitor 📞🖥️

![Asterisk Monitor](https://img.shields.io/badge/Asterisk-Monitor-blue)
![Go](https://img.shields.io/badge/Go-1.21%2B-00ADD8)
![License](https://img.shields.io/badge/License-MIT-green)
![Platform](https://img.shields.io/badge/Platform-Linux-lightgrey)

Мощный терминальный монитор для Asterisk PBX с красивым TUI интерфейсом, написанный на Go. Предоставляет полный контроль над вашей Asterisk системой через интуитивно понятный интерфейс.

## ✨ Возможности

### 📊 **Дашборд**
- Мониторинг состояния системы в реальном времени
- Метрики производительности (CPU, память, диск)
- Статус SIP пиров и активных вызовов
- Время работы системы и нагрузка

### 🔍 **Диагностика**
- Быстрая проверка состояния сервисов
- Полная диагностика системы
- Проверка сетевых подключений
- Валидация конфигураций

### 📞 **Каналы**
- Просмотр активных каналов в реальном времени
- Информация о состоянии вызовов
- Детали Caller ID и продолжительности
- Автоматическое обновление

### 📋 **Логи**
- Просмотр логов Asterisk с фильтрацией
- Фильтры по уровню (ERROR, WARNING, DEBUG)
- Поиск по ключевым словам
- Настраиваемое количество строк

### 🛡️ **Безопасность**
- Сканирование безопасности системы
- Проверка открытых портов
- Анализ конфигураций безопасности
- Рекомендации по улучшению

### 💾 **Бэкапы**
- Создание резервных копий конфигураций
- Восстановление из бэкапов
- Управление архивом бэкапов
- Автоматическое именование с timestamp

### ⚙️ **Настройки**
- Конфигурация подключения к Asterisk
- Настройки мониторинга
- Параметры безопасности
- Интервалы обновления

## 🚀 Быстрый старт

### Предварительные требования

- **Asterisk** 13+ (установленный и настроенный)
- **Go** 1.21 или новее
- **Linux** система

### Установка Asterisk

**Ubuntu/Debian:**
```bash
sudo apt update
sudo apt install asterisk asterisk-dev
sudo systemctl enable asterisk
sudo systemctl start asterisk
```

**CentOS/RHEL:**
```bash
sudo yum install epel-release
sudo yum install asterisk asterisk-devel
sudo systemctl enable asterisk
sudo systemctl start asterisk
```

### Установка приложения

1. **Клонируйте репозиторий:**
```bash
git clone https://github.com/your-username/asterisk-monitor.git
cd asterisk-monitor
```

2. **Установите зависимости:**
```bash
go mod download
```

3. **Соберите приложение:**
```bash
go build -o asterisk-monitor main.go
```

4. **Запустите:**
```bash
./asterisk-monitor
```

## ⚙️ Конфигурация

### Настройка AMI (Asterisk Manager Interface)

Добавьте в `/etc/asterisk/manager.conf`:

```ini
[general]
enabled = yes
port = 5038
bindaddr = 127.0.0.1

[admin]
secret = amp111
deny = 0.0.0.0/0.0.0.0
permit = 127.0.0.1/255.255.255.0
read = system,call,log,verbose,command,agent,user,config,command,dtmf,reporting,cdr,dialplan
write = system,call,log,verbose,command,agent,user,config,command,dtmf,reporting,cdr,dialplan
```

Перезагрузите Asterisk:
```bash
sudo systemctl restart asterisk
```

### Автоматическая настройка

Запустите приложение - конфигурационный файл создастся автоматически в:
`~/.asterisk-monitor/config.ini`

## 🎯 Использование

### Навигация
- **1-7** - Переключение между модулями
- **Q** или **Ctrl+C** - Выход
- **R** - Обновить данные (в большинстве модулей)
- **TAB** - Переключение между полями ввода

### Модули

1. **📊 Дашборд** - Основная информация о системе
2. **🔍 Диагностика** - Проверка состояния системы
3. **📞 Каналы** - Активные вызовы и каналы
4. **📋 Логи** - Просмотр логов Asterisk
5. **🛡️ Безопасность** - Сканирование безопасности
6. **💾 Бэкапы** - Резервное копирование и восстановление
7. **⚙️ Настройки** - Конфигурация приложения

## 🔧 Расширенная установка

### Systemd сервис (рекомендуется)

Создайте файл `/etc/systemd/system/asterisk-monitor.service`:

```ini
[Unit]
Description=Asterisk Monitor
After=network.target asterisk.service
Wants=asterisk.service

[Service]
Type=simple
User=asterisk
WorkingDirectory=/opt/asterisk-monitor
ExecStart=/opt/asterisk-monitor/asterisk-monitor
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Настройте сервис:
```bash
sudo mkdir -p /opt/asterisk-monitor
sudo cp asterisk-monitor /opt/asterisk-monitor/
sudo systemctl daemon-reload
sudo systemctl enable asterisk-monitor
sudo systemctl start asterisk-monitor
```

### Скрипт автоматической установки

```bash
chmod +x deploy.sh
sudo ./deploy.sh
```

## 🗂️ Структура проекта

```
asterisk-monitor/
├── main.go                 # Основной файл приложения
├── config/
│   └── config.go          # Управление конфигурацией
├── monitors/
│   └── linux.go           # Мониторинг для Linux систем
├── types/
│   └── types.go           # Структуры данных
├── ui/
│   ├── common.go          # Общие UI компоненты
│   ├── dashboard.go       # Дашборд
│   ├── diagnostics.go     # Диагностика
│   ├── channels.go        # Активные каналы
│   ├── logs.go           # Просмотр логов
│   ├── security.go       # Безопасность
│   ├── backup.go         # Бэкапы
│   └── settings.go       # Настройки
└── README.md
```

## 🛠️ Разработка

### Зависимости

Основные зависимости:
- **[BubbleTea](https://github.com/charmbracelet/bubbletea)** - TUI фреймворк
- **[LipGloss](https://github.com/charmbracelet/lipgloss)** - Стилизация TUI
- **[ini](https://github.com/go-ini/ini)** - Парсинг INI файлов

### Сборка для разработки

```bash
# Установите зависимости
go mod tidy

# Запустите в режиме разработки
go run main.go

# Соберите с оптимизациями
go build -ldflags="-s -w" -o asterisk-monitor main.go
```

### Тестирование

```bash
# Запустите тесты
go test ./...

# Проверка покрытия
go test -cover ./...
```

## 🔒 Безопасность

### Рекомендации по безопасности

1. **AMI доступ** - Ограничьте доступ к порту 5038 только localhost
2. **Пользователь** - Запускайте приложение под отдельным пользователем
3. **Бэкапы** - Храните бэкапы в защищенном месте
4. **Логи** - Регулярно проверяйте логи на подозрительную активность

### Настройка sudo прав

Добавьте в `/etc/sudoers`:
```
asterisk-monitor ALL=(ALL) NOPASSWD: /bin/systemctl status asterisk, /bin/systemctl restart asterisk
asterisk-monitor ALL=(ALL) NOPASSWD: /usr/bin/tail /var/log/asterisk/*
```

## 🐛 Troubleshooting

### Common Issues

**Проблема**: "Asterisk не установлен"
```bash
# Решение: Установите Asterisk
sudo apt install asterisk
```

**Проблема**: "Нет доступа к AMI"
```bash
# Решение: Проверьте manager.conf и перезагрузите Asterisk
sudo systemctl restart asterisk
```

**Проблема**: "Недостаточно прав"
```bash
# Решение: Запустите с правами asterisk пользователя
sudo -u asterisk ./asterisk-monitor
```

### Логи

Просмотр логов приложения:
```bash
sudo journalctl -u asterisk-monitor -f
```

Логи Asterisk:
```bash
sudo tail -f /var/log/asterisk/messages
```

## 🤝 Вклад в проект

Мы приветствуем вклады! Пожалуйста:

1. Форкните репозиторий
2. Создайте feature ветку (`git checkout -b feature/amazing-feature`)
3. Закоммитьте изменения (`git commit -m 'Add amazing feature'`)
4. Запушьте ветку (`git push origin feature/amazing-feature`)
5. Откройте Pull Request

## 📄 Лицензия

Этот проект распространяется под MIT License - смотрите файл [LICENSE](LICENSE) для деталей.

## 🙏 Благодарности

- [Charmbracelet](https://charmbracelet.com/) за отличные TUI библиотеки
- Команда Asterisk за прекрасную PBX систему
- Сообществу Go за мощный инструментарий

---

**Asterisk Monitor** - сделайте управление вашей Asterisk системой простым и эффективным! 🚀

Для вопросов и поддержки создавайте issue в репозитории проекта.