# libreta

Хранилка заметок.

Заметки хранятся в sqlite БД.

Приложение на Go предоставляет API и Web-интерфейс.

Web-интерфейс встроен прямо в исполняемый файл.

Возможность вызывать API прямо из командной строки:
    ./app --db notes.db api-call methodName param1

[WYSIWYM](https://en.wikipedia.org/wiki/WYSIWYM) редактор для заметок.

## Название

* libreta — испанский, блокнот.

## План:

* Go приложение с базовым API и работой с БД.
  * Go Hello World.
  * API Hello World.
  * Открытие БД.
  * MVP схема БД.
  * API для чтения записей.
  * API для сохранения записей.
* Web интерфейс.

## Референсы и источники вдохновения

* [Embedding Vue.js Apps in Go](https://hackandsla.sh/posts/2021-06-18-embed-vuejs-in-go/)
* [How To Corrupt An SQLite Database File](https://www.sqlite.org/howtocorrupt.html)
* [Bahaviour of st on sqlite files in various scenarios - Support - Syncthing Community Forum](https://forum.syncthing.net/t/bahaviour-of-st-on-sqlite-files-in-various-scenarios/5921)
* [brettkromkamp/awesome-knowledge-management: A curated list of amazingly awesome articles, people, applications, software libraries and projects related to the knowledge management space](https://github.com/brettkromkamp/awesome-knowledge-management)
* [Fossil: Home](https://www.fossil-scm.org/home/doc/trunk/www/index.wiki)
* [Recfiles - Wikipedia](https://en.wikipedia.org/wiki/Recfiles)
* [GitHub - benweet/stackedit: In-browser Markdown editor](https://github.com/benweet/stackedit)
* [GitHub - nhn/tui.editor: Markdown WYSIWYG Editor. GFM Standard + Chart & UML Extensible.](https://github.com/nhn/tui.editor)
* [JavaScript Markdown Editor - SimpleMDE](https://simplemde.com/)
* [GitHub - Ionaru/easy-markdown-editor: EasyMDE: A simple, beautiful, and embeddable JavaScript Markdown editor. Delightful editing for beginners and experts alike. Features built-in autosaving and spell checking.](https://github.com/Ionaru/easy-markdown-editor)
* [Milkdown](https://milkdown.dev/#/) Plugin Based WYSIWYG Markdown Editor Framework
* [ProseMirror](https://prosemirror.net/)
* [Replit - Ace, CodeMirror, and Monaco: A Comparison of the Code Editors You Use in the Browser](https://blog.replit.com/code-editors)
* [Editor.js](https://editorjs.io/)

* [SilverBullet](https://silverbullet.md/)

## Наблюдения

FTS5 в Sqlite3 в индексе с [External Content Tables](https://www.sqlite.org/fts5.html#external_content_tables) необходим rowid, который может быть
только INTEGER. Неявный `rowid` [может внезапно измениться](https://www.sqlite.org/rowidtable.html) в результате работы `VACUUM`, поэтому его необходимо
зафиксировать в таблице как `INTEGER PRIMARY KEY`.

## Поток сознания

Обратные ссылки — must have.

Записи-закладки — URL, теги, описание. Вставка ссылок в заметки через ID закладки. Синхронизация с браузером, веб-архивом, и т.п.

Микрозаметки. Как сообщение в мессенджере, просто время и текст/картинка.

Списки. Список микрозаметок, как набор сообщений в мессенджере. Автогенерируемый список артефактов по поисковому запросу (тексту, тегам и т.п.).

Экспорт списка заметок как блога.

Синхронизируемые (с внешними системами) списки. Плейлисты, вотчлисты, и т.п.

Списки — частные случаи деревьев?

Цветовые маркеты (фон карточки, фон заголовка). Через специальные теги? Отображение легенды — квадратики с цветами и подписанным смыслом. Иконки? Иконки-эмодзи?

Content adressable записи? SHA хеш вместо ID?

ID заметок (артефактов) из временных меток? [Zettelkasten](https://zettelkasten.de/posts/add-identity/)

Проблему с синхронизацией SQLite через Syncthing можно попробовать обойти создавая отдельные бэкапы в синхронизированой директории, каждый из которых именуется по имени конкретного устройства, затем инстансы запущенные на других устройствах читают все бэкапы и мерджат в рабочую БД все новые изменения.

UI. Добавление ссылки на другой артефакт. Выделяешь текст, появляется плавающее окно поиска, в котором находишь нужный и выбираешь его как цель для ссылки.

UI. Добавление ссылки на ещё не существующую микрозаметку. Во всплывающем окне поиска цели ссылки можно переключиться в режим редактора, в котором можно тут же ввести текст новой заметки, после чего ссылка на неё вставится в оригинальную.

UI. Всплывающие окна при наведении на ссылку с превью содержимого ссылки. Как тут: https://notes.andymatuschak.org/About_these_notes

Заметки комментарии. Чётко связаны с каким-то другим артефактом (заметкой, ссылкой, списком и т.п.). Может быть реализовать как просто один из множества видов связей между артефактами.

Автоматический сбор метаинформации при создании новой заметки: время, место, GPS?, устройство и т.п.

UI. Артефакты отображаются в виде карточек, у разных типов артефактов могут быть разные отображения карточек (заметка, ссылка, картинка). Карточки могут вкладываться друг в друга.

UI. Ссылка на артефакт-ссылку из артефакта заметки может отображаться как конечная цель этого артефакта-ссылки.

Артефакт-ссылка может иметь несколько альтернативных целей (например на оригинал и на web-архив).

Три формы организации:
* Дерево. Записи могут формировать дерево, как дерево комментариев.
* Ссылки. Записи могут ссылаться на другие, образуя двунаправленную связь. Должна быть возможность посмотреть список обратных ссылок.
* Каналы. Что-то вроде тегов или тем. Запись может быть помечена несколькими тегами, например: `#dev #db`.

Навигация:
* Строка поиска. Поддержка сложных запросов с фильтрацией по аттрибутам (дата создания, изменения, теги и т.п.).
* Таймлайн. Открывает все записи в хронологическом порядке.
* Закладки. В качестве закладки может выступать ссылка на конкретную запись или поисковый запрос. Закладки можно оформлять: добавлять иконку, менять название, цвет и т.п.
