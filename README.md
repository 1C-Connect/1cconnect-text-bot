# Connect Text Bot

Данный бот реализует произвольное конфигурироемое текстовое меню на заданных линиях поддержки.

## Требования к окружению

* OS: Linux/Windows
* Go: 1.22+

## Сборка и запуск

### Сборка из исходников

```bash
./build.sh
```

**Note:** Сборка требует установленного окружения!

### Запуск собранной версии

```bash
./connect-text-bot --config=config.yml --bot=bot.yml
```

Где:

* `--config` - путь к конфигу (путь по умолчанию - `./config/config.yml`).
* `--bot` - путь к конфигу бота (путь по умолчанию - `./config/bot.yml`).

**Note:** Бот остлеживает изменения конфигураци меню, содержимое можно менять на горячую, но стоит преварительно
проверять через валидатор (например https://onlineyamltools.com/validate-yaml)

### Разворачивание бота

Для того чтобы бот работал корректно необходимо выполнить следующие требования и действия:

* Необходимо подготовить машину имеющую доступ в интернет и способную принимать HTTP запросы из интернета
  * Лучшим выбором будет **Linux**, возможно использование виртуальной машины
* Необходим полный тариф https://1c-connect.com/ru/forpartners/#2
* Настроить пользователя API в учетной системе 1С-Коннект
  * Раздел Администрирование -> Настройки API
  * Создаете нового пользователя
* Включить **Внешний сервер для обработки данных** в нужной линии
  * Откройте в УС карточку линии и в разделе **Чат-бот** включите соотвествующую настройку
* Необходимо получить ID линии для которой была включена внешняя обработка
  * Выполнит запрос https://1c-connect.atlassian.net/wiki/spaces/PUBLIC/pages/2156429313/v1+line (можно открыть ссылку https://push.1c-connect.com/v1/line/ в браузере и ввести логин/пароль от ранее созданного пользователя)
  * Найти линию в списке и сохранить ее ID
* На подготовленный сервер загрузить приложение бота, файл с меню
* Сконфигурировать и запустить приложение
  * Создать конфигурационный файл. Пример лежит в файле `config/config.yml.sample` и отредактировать его
  * Указать в блоке **server** адрес к серверу на котором развернут бот
    * **Note:** Помните что указанный хост и порт должны быть доступны из сети Интернет
  * Указать логин/пароль ранее созданного пользователя API
  * Указать ID линии в разделе **lines**, можно указывать несколько линий
  * Бот может отправлять файлы, в конфигурационном файле можно указать путь к папке с файлами, далее в меню указывать имена файлов для отправки в чат
  * Приложение может быть запущено с указание путей к соответсвующим файлам

## Конфигурация меню

Конфигурационный файл представляет собой `yml` файл вида:

```yaml
use_qna:
  enabled: true

menus:
  start:
    answer:
      - chat: 'Здравствуйте.'
    buttons:
      - button:
          id: 1
          text: 'a'
      - button:
          id: 2
          text: 'b'
      - button:
          id: 3
          text: 'nested'
          menu:
            id: 'nested_menu'
            answer:
              - chat: 'Welcome to nested menu.'
            buttons:
              - button:
                  id: 1
                  text: 'get information'
                  chat:
                    - chat: 'information'
              - button:
                  back_button: true
      - button:
          id: 4
          text: 'send file'
          chat:
            - file: 'file.pdf'
              file_text: 'you received file!'
  final_menu:
    answer:
      - chat: 'Могу ли я вам чем-то еще помочь?'
    buttons:
      - button:
          id: 1
          text: 'Да'
          goto: 'start'
      - button:
          id: 2
          text: 'Нет'
          chat:
            - chat: 'Спасибо за обращение!'
          close_button: true
      - button:
          redirect_button: true

back_button:
  id: 8
  text: 'Назад'

redirect_button:
  id: 0
  text: 'Соединить со специалистом'

close_button:
  id: 9
  text: 'Закрыть обращение'

error_message: 'Команда неизвестна. Попробуйте еще раз'
```

**Note:** Директория с файлами задается параметром `files_dir`, в конфигурационном файле программы `config.yml`.

Конфигурация состоит из различных меню. Меню `start` - появляется после первого сообщения от пользователя. `final_menu` - резюмирует диалог.

Каждое меню состоит из блоков `answer` и `buttons`.

Блок `answer` отвечает за сообщение при переходе на данный раздел.

При переходе между меню есть возможность отправить текст:

```yaml
menus:
  start:
    answer:
      - chat: 'Здравствуйте.'
    buttons:
    ...
```

Или файл:

```yaml
menus:
  start:
    answer:
      - file: 'file.pdf'
        file_text: 'Сопроводительное письмо к файлу.'
    buttons:
    ...
```

Или несколько сообщений и файлов:

```yaml
menus:
  start:
    answer:
      - chat: 'Сообщение 1'
      - chat: 'Сообщение 2'
      - file: 'file1.pdf'
      - file: 'file2.pdf'
      - chat: 'Сообщение 3'
    buttons:
    ...
```

Также при нажатии на кнопку есть возможность отправить несколько сообщений или файлов:

```yaml
menus:
  start:
    answer:
      - chat: 'Сообщение 1'
    buttons:
      - button:
          id: 1
          text: 'Кнопка 1'
          chat:
            - chat: 'Сообщение 1'
            - chat: 'Сообщение 2'
            - file: 'file1.pdf'
              file_text: 'Сопроводительное письмо к файлу1'
            - file: 'file2.pdf'
              file_text: 'Сопроводительное письмо к файлу2'
    ...
```

Блок `buttons` - представляет собой список кнопок на данном уровне. У кнопки обязательно должен текст `text`.

```yaml
buttons:
  - button:
      id: 1
      text: 'Текст кнопки' # обязательное поле
```

Если у кнопки нет пункта `goto`, то после нажатия на неё будет совершен переход в `final_menu`.

```yaml
buttons:
  - button:
      id: 1 # Нажатие на эту кнопку переведёт в final_menu
      text: 'Кнопка 1'
  - button:
      id: 2 # Нажатие на эту кнопку переведёт в some_menu
      text: 'Кнопка 2'
      goto: 'some_menu' # Явно указываем в какое меню переведет кнопка
```

Если у кнопки есть пункт `menu`, то после нажатия на неё будет совершен переход в подменю.

```yaml
buttons:
  - button:
      id: 1 # Нажатие на эту кнопку переведёт в nested_menu
      text: 'Текст кнопки'
      chat: 'Сообщение'
      menu:
        id: 'nested_menu'
        ...
```

### Настройки по умолчанию

Для специальных пунктов меню:

`back_button` - описывает кнопку "Назад", которая переводит меню на уровень назад.

`close_button` - описывает кнопку "Закрыть обращение", которая завершает работу с обращением.

`redirect_button` - описывает кнопку "Перевести на специалиста", которая переводит работу из бот-мению на свободного
специалиста или ставит обращение в очередь, если нет свободных специалистов.

Можно задать описания по умолчанию:

```yaml
back_button:
  id: 8
  text: 'Назад'

redirect_button:
  id: 0
  text: 'Соединить со специалистом'

close_button:
  id: 9
  text: 'Закрыть обращение'
```

Если в конфиге отсутствует `final_menu`, будет использовано меню по умолчанию:

```yaml
menus:
...
  final_menu:
    answer:
      - chat: 'Могу ли я вам чем-то еще помочь?'
    buttons:
      - button:
          id: 1
          text: 'Да'
          goto: 'start'
      - button:
          id: 2
          text: 'Нет'
          chat:
            - chat: 'Спасибо за обращение!'
          close_button: true
      - button:
          redirect_button: true
...
```

Если в конфиге отсутствует `error_message`, будет использовано сообщение по умолчанию:

```yaml
error_message: 'Команда неизвестна. Попробуйте еще раз'
```

Можно сделать сделать так, чтобы бот здоровался только один раз.

Для этого необходимо добавить следующую строчку в конфиг бота (файл `bot.yml`):

```yaml
first_greeting: true
```

А также задать текст приветственного сообщения (файл `bot.yml`):
```yaml
greeting_message: 'Здравствуйте.'
```

### Как отправить текст

```yaml
buttons:
  - button:
      id: 1
      text: 'Текст кнопки'
      chat:
        - chat: 'Сообщение'
```

### Как отправить файл

```yaml
buttons:
  - button:
      id: 1
      text: 'Текст кнопки'
      chat:
        - file: 'file.pdf'
          file_text: 'Сопроводительное сообщение к файлу.'
```

### Как закрыть обращение

```yaml
buttons:
  - button:
      id: 9
      text: 'Закрыть обращение'
      close_button: true
```

### Как перевести на свободного специалиста

```yaml
buttons:
  - button:
      id: 0
      text: 'Перевести на свободного специалиста'
      redirect_button: true
```

### Как перевести на конкретного специалиста

```yaml
buttons:
  - button:
      id: 2
      text: 'Соединить со специалистом Иванов И.И.'
      appoint_spec_button: bb296731-3d58-4c4a-8227-315bdc2bf3ff
```

### Как перевести на случайного специалиста из списка

```yaml
buttons:
  - button:
      id: 2
      text: 'Соединить с одним из консультантов'
      appoint_random_spec_from_list_button:
            - bb296731-3d58-4c4a-8227-315bdc2bf1ff
            - bb296731-3d58-4c4a-8227-315bdc2bf2ff
            - bb296731-3d58-4c4a-8227-315bdc2bf3ff
```

### Как перевести обращение на другую линию

```yaml
buttons:
  - button:
      id: 3
      text: 'Перевод обращения на линию "1С-Коннект: Техподдержка"'
      reroute_button: bb296731-3d58-4c4a-8227-315bdc2bf3ff
```

### Как выполнить команду на стороне сервера

```yaml
buttons:
  - button:
      id: 3
      text: 'Выполнить команду на стороне сервера'
      exec_button: "./scripts/example.sh {{ .UserId }} {{ .Surname }} {{ .Name }}"
```

В команду можно передать данный относящиеся к структуре объекта User (Пользователь) в формате `{{ .НазваниеПоля }}`

Cкрипт `example.sh` имеет следующее содержание

```bash
#!/bin/bash

echo -n $1 | base64 
echo -n $2 $3
```

Не забудьте сделать скрипт исполняемым

```bash
chmod +x ./scripts/example.sh
```

### Как создать меню

#### Способ №1

```yaml
menus:
  start:
    answer:
      - chat: 'Здравствуйте.'
    buttons:
      - button:
          id: 1
          text: 'a'
          menu:
            id: 'новое_меню'
            answer:
              - chat: 'welcome'
            buttons:
              - button:
                  id: 1
                  text: 'Текст кнопки'
  final_menu:
    answer:
      - chat: 'Могу ли я вам чем-то еще помочь?'
    buttons:
      - button:
          id: 1
          text: 'Да'
          goto: 'start'
      - button:
          id: 2
          text: 'Нет'
          chat:
            - chat: 'Спасибо за обращение!'
          close_button: true
      - button:
          redirect_button: true
back_button:
  id: 8
  text: 'Назад'

redirect_button:
  id: 0
  text: 'Соединить со специалистом'

close_button:
  id: 9
  text: 'Закрыть обращение'
```

#### Способ №2

```yaml
menus:
  start:
    answer:
      - chat: 'Здравствуйте.'
    buttons:
      - button:
          id: 1
          text: 'a'
          goto: 'новое_меню'

  новое_меню:
    answer:
      - chat: 'welcome'
    buttons:
      - button:
          id: 1
          text: 'Текст кнопки'

  final_menu:
    answer:
      - chat: 'Могу ли я вам чем-то еще помочь?'
    buttons:
      - button:
          id: 1
          text: 'Да'
          goto: 'start'
      - button:
          id: 2
          text: 'Нет'
          chat:
            - chat: 'Спасибо за обращение!'
          close_button: true
      - button:
          redirect_button: true
back_button:
  id: 8
  text: 'Назад'

redirect_button:
  id: 0
  text: 'Соединить со специалистом'

close_button:
  id: 9
  text: 'Закрыть обращение'
```

## Использование подсказок из баз знаний

### Глобальные параметры использования база знаний компании задаются разделом `use_qna`

```yaml
use_qna:
  enabled: true
```

`enabled` - включен поиск ответов в базах знаний

### В конкретном меню можно отключить использование подсказок, воспользовавшись параметром `qna_disable`

```yaml
...
      menu:
        ...
        qna_disable: true
...
```

Если в конфиге отсутствует `fail_qna_menu`, будет использовано меню по умолчанию в случае отсутcвия ответа на произвольный вопрос:

```yaml
menus:
...
  fail_qna_menu:
    answer:
      - chat: |
          Я Вас не понимаю.

          Попробуете еще раз или перевести обращение на специалиста?
    buttons:
      - button:
          id: 1
          text: 'Продолжить'
          back_button: true
      - button:
          id: 2
          text: 'Закрыть обращение'
          chat:
            - chat: 'Спасибо за обращение!'
          close_button: true
      - button:
          redirect_button: true
...
```
