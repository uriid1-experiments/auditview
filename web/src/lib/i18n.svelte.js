// UI language. English is the default; the choice lives in localStorage
// and every t()/data()/fmtTime() call re-evaluates when it changes,
// because they read reactive state.

const STORAGE_KEY = 'av_lang'

export const LANGS = [
  ['en', 'English'],
  ['ru', 'Русский'],
]

const strings = {
  en: {
    'nav.events': 'Events',
    'nav.rules': 'Rules',
    'nav.alerts': 'Alerts',
    'settings.title': 'Settings',
    'settings.language': 'Language',

    'common.cancel': 'Cancel',
    'common.error': 'Error',

    'col.time': 'time',
    'col.key': 'key',
    'col.event': 'event',
    'col.process': 'process',
    'col.result': 'result',
    'col.rule': 'rule',
    'col.object': 'object',
    'col.delivery': 'delivery',
    'col.hint': 'drag the right edge; double-click resets widths',

    'events.stored': '{n} events stored',
    'events.found': 'found: {n}',
    'events.pause': '⏸ pause',
    'events.live': '▶ live',
    'events.search': 'search (path, command, user…)',
    'events.exeFilter': 'filter by exe',
    'events.resultAny': 'result: any',
    'events.keyedOnly': 'rules only',
    'events.reset': 'reset',
    'events.apiError': 'API error: {error}. Is the server running? (sudo ./auditview)',
    'events.more': 'show more ({n})',
    'events.empty': 'No events match the current filters',

    'row.muteAdded': '✓ Added to {name}. Next: Rules tab → Apply rules.',
    'row.error': 'Error: {error}',
    'row.rawFailed': '(could not load raw records)',
    'row.process': 'Process',
    'row.user': 'User',
    'row.auid': 'auid (login user)',
    'row.syscall': 'Syscall',
    'row.call': 'call',
    'row.result': 'result',
    'row.files': 'Files',
    'row.path': 'path',
    'row.type': 'type',
    'row.owner': 'owner',
    'row.raw': 'Raw records',
    'row.mute': '🔇 Mute',
    'row.muteHint':
      'This program will no longer produce events for this path (other programs still will). You can edit the line:',
    'row.addExclusion': 'Add exclusion',

    'rules.auditOn': 'audit: on',
    'rules.auditOff': 'audit: off',
    'rules.lost': 'lost: {n}',
    'rules.inSync': 'files = kernel',
    'rules.outOfSync': 'files ≠ kernel (apply needed)',
    'rules.expert': 'expert mode',
    'rules.saveFirstTitle': 'Save the modified files first',
    'rules.applyTitle': 'augenrules --load: reloads ALL rules from rules.d into the kernel',
    'rules.applying': 'applying…',
    'rules.saveFirst': 'Save first',
    'rules.applyConfirm': 'Really apply? (click again)',
    'rules.apply': 'Apply rules',
    'rules.unsaved': 'Unsaved',
    'rules.unsavedNote':
      'Edits live only in the browser: “Apply” loads the on-disk version into the kernel, without them.',
    'rules.saving': 'saving…',
    'rules.saveAll': '💾 Save all',
    'rules.save': '💾 Save',
    'rules.unsavedBadge': 'unsaved',
    'rules.legend': 'auditd flag cheat sheet',
    'rules.applied': 'Applied',
    'rules.applyFailed': 'Apply failed',
    'rules.loadedCount': 'rules loaded: {n}',
    'rules.addLine': '+ line',
    'rules.addRule': '+ rule',
    'rules.syntaxProblems': 'Syntax problems — auditd will reject these lines:',
    'rules.deleteLine': 'delete line',
    'rules.newFile': 'new file, e.g. 10-docker.rules',
    'rules.createFile': 'Create file',

    'card.clickToEdit': 'click to edit',
    'card.expertOnly': 'editable only in expert mode',
    'card.deleteRule': 'delete rule',
    'card.recipe': '— recipe (optional) —',
    'card.watch': '👁 Watch',
    'card.never': '🔇 Exclusion (no events)',
    'card.path': 'path',
    'card.exact': 'exact path',
    'card.dir': 'directory + contents',
    'card.catch': 'catch:',
    'card.keyPh': 'key, e.g. cred_npm',
    'card.exePh': 'program (empty = any), e.g. /usr/bin/konsole',
    'card.done': 'Done',

    'perm.r': 'read',
    'perm.w': 'write',
    'perm.x': 'execute',
    'perm.a': 'attributes',
    'lang.wholeDir': ' (whole directory)',
    'lang.anyProgram': 'any program',
    'lang.anyAccess': 'any access',
    'lang.noEvents': 'No events: {who} — {acts}: {path}{scope}',
    'lang.watch': 'Watch: {acts} — {path}{scope}',
    'noise.header': '# Legitimate-noise suppression (never rules, loaded first)',

    'alerts.desktop': 'desktop',
    'alerts.command': 'command',
    'alerts.on': 'on',
    'alerts.off': 'off',
    'alerts.sending': 'sending…',
    'alerts.test': 'Send test notification',
    'alerts.rulesSummary': 'Alert rules ({n} active) — edited in /etc/auditview/alerts.json',
    'alerts.keys': 'keys:',
    'alerts.cooldown': 'cooldown {n}s',
    'alerts.exceptExe': 'except exe:',
    'alerts.disabled': 'disabled',
    'alerts.empty': 'No alerts fired yet',
  },
  ru: {
    'nav.events': 'События',
    'nav.rules': 'Правила',
    'nav.alerts': 'Алерты',
    'settings.title': 'Настройки',
    'settings.language': 'Язык',

    'common.cancel': 'Отмена',
    'common.error': 'Ошибка',

    'col.time': 'время',
    'col.key': 'ключ',
    'col.event': 'событие',
    'col.process': 'процесс',
    'col.result': 'результат',
    'col.rule': 'правило',
    'col.object': 'объект',
    'col.delivery': 'доставка',
    'col.hint': 'тяните за правую границу; двойной клик — сброс ширин',

    'events.stored': '{n} событий в буфере',
    'events.found': 'найдено: {n}',
    'events.pause': '⏸ пауза',
    'events.live': '▶ live',
    'events.search': 'поиск (путь, команда, пользователь…)',
    'events.exeFilter': 'фильтр по exe',
    'events.resultAny': 'результат: любой',
    'events.keyedOnly': 'только правила',
    'events.reset': 'сбросить',
    'events.apiError': 'Ошибка API: {error}. Сервер запущен? (sudo ./auditview)',
    'events.more': 'показать ещё ({n})',
    'events.empty': 'Нет событий по текущим фильтрам',

    'row.muteAdded': '✓ Добавлено в {name}. Осталось: вкладка «Правила» → «Применить правила».',
    'row.error': 'Ошибка: {error}',
    'row.rawFailed': '(не удалось загрузить сырые записи)',
    'row.process': 'Процесс',
    'row.user': 'Пользователь',
    'row.auid': 'auid (кто залогинился)',
    'row.syscall': 'Syscall',
    'row.call': 'вызов',
    'row.result': 'результат',
    'row.files': 'Файлы',
    'row.path': 'путь',
    'row.type': 'тип',
    'row.owner': 'владелец',
    'row.raw': 'Сырые записи',
    'row.mute': '🔇 Заглушить',
    'row.muteHint':
      'Эта программа больше не будет создавать события по этому пути (другие программы — будут). Строку можно поправить:',
    'row.addExclusion': 'Добавить исключение',

    'rules.auditOn': 'аудит: включён',
    'rules.auditOff': 'аудит: выключен',
    'rules.lost': 'lost: {n}',
    'rules.inSync': 'файлы = ядро',
    'rules.outOfSync': 'файлы ≠ ядро (нужно применить)',
    'rules.expert': 'режим эксперта',
    'rules.saveFirstTitle': 'Сначала сохраните изменённые файлы',
    'rules.applyTitle': 'augenrules --load: перезагрузит ВСЕ правила из rules.d в ядро',
    'rules.applying': 'применяется…',
    'rules.saveFirst': 'Сначала сохраните',
    'rules.applyConfirm': 'Точно применить? (клик ещё раз)',
    'rules.apply': 'Применить правила',
    'rules.unsaved': 'Не сохранено',
    'rules.unsavedNote':
      'Правки живут только в браузере: «Применить» загрузит в ядро версию с диска, без них.',
    'rules.saving': 'сохраняем…',
    'rules.saveAll': '💾 Сохранить всё',
    'rules.save': '💾 Сохранить',
    'rules.unsavedBadge': 'не сохранено',
    'rules.legend': 'Шпаргалка по флагам auditd',
    'rules.applied': 'Применено',
    'rules.applyFailed': 'Ошибка применения',
    'rules.loadedCount': 'загружено правил: {n}',
    'rules.addLine': '+ строка',
    'rules.addRule': '+ правило',
    'rules.syntaxProblems': 'Проблемы в синтаксисе — auditd такие строки не примет:',
    'rules.deleteLine': 'удалить строку',
    'rules.newFile': 'новый файл, например 10-docker.rules',
    'rules.createFile': 'Создать файл',

    'card.clickToEdit': 'нажмите, чтобы изменить',
    'card.expertOnly': 'правится только в режиме эксперта',
    'card.deleteRule': 'удалить правило',
    'card.recipe': '— рецепт (не обязательно) —',
    'card.watch': '👁 Следить',
    'card.never': '🔇 Исключение (без событий)',
    'card.path': 'путь',
    'card.exact': 'точный путь',
    'card.dir': 'каталог + содержимое',
    'card.catch': 'ловить:',
    'card.keyPh': 'метка, напр. cred_npm',
    'card.exePh': 'программа (пусто = любая), напр. /usr/bin/konsole',
    'card.done': 'Готово',

    'perm.r': 'чтение',
    'perm.w': 'изменение',
    'perm.x': 'запуск',
    'perm.a': 'атрибуты',
    'lang.wholeDir': ' (весь каталог)',
    'lang.anyProgram': 'любая программа',
    'lang.anyAccess': 'любое обращение',
    'lang.noEvents': 'Без событий: {who} — {acts}: {path}{scope}',
    'lang.watch': 'Следить: {acts} — {path}{scope}',
    'noise.header': '# Подавление легитимного шума (never-правила, грузятся первыми)',

    'alerts.desktop': 'рабочий стол',
    'alerts.command': 'команда',
    'alerts.on': 'вкл',
    'alerts.off': 'выкл',
    'alerts.sending': 'отправляется…',
    'alerts.test': 'Тестовое уведомление',
    'alerts.rulesSummary': 'Правила алертинга ({n} активных) — правятся в /etc/auditview/alerts.json',
    'alerts.keys': 'ключи:',
    'alerts.cooldown': 'кулдаун {n}с',
    'alerts.exceptExe': 'кроме exe:',
    'alerts.disabled': 'выключено',
    'alerts.empty': 'Срабатываний пока не было',
  },
}

// Structured (non-string) translations: tables shown as-is.
const tables = {
  en: {
    legend: [
      ['-w PATH', 'watch this file/directory'],
      ['-p rwxa', 'which accesses to catch: r read, w write, x execute, a attributes'],
      ['-k KEY', 'event label (key) — filters and alerts match on it'],
      ['-a always,exit', 'produce an event (a regular watch rule)'],
      ['-a never,exit', 'do NOT produce an event — an exclusion; must precede the always rules'],
      ['-F dir=PATH', 'a directory with everything inside it'],
      ['-F path=PATH', 'exactly this object (a file, or the directory itself without its contents)'],
      ['-F perm=rwxa', 'same as -p, for the -a form'],
      ['-F exe=PATH', 'only when this program is the actor'],
      ['-S syscall', 'which system calls (all — any)'],
    ],
    recipes: [
      'Secret read (file)',
      'Secret reads (directory)',
      'Config changes (directory)',
      'Docker credentials',
      'npm token',
      'kubeconfig',
      'GPG keys (directory)',
      'Mute a program for a path',
    ],
  },
  ru: {
    legend: [
      ['-w ПУТЬ', 'следить за этим файлом/каталогом'],
      ['-p rwxa', 'какие действия ловить: r чтение, w изменение, x запуск, a атрибуты'],
      ['-k МЕТКА', 'метка события (ключ) — по ней фильтры и алерты'],
      ['-a always,exit', 'создавать событие (обычное правило слежения)'],
      ['-a never,exit', 'НЕ создавать событие — исключение; должно стоять выше always-правил'],
      ['-F dir=ПУТЬ', 'каталог со всем содержимым'],
      ['-F path=ПУТЬ', 'ровно этот объект (файл или сам каталог, без содержимого)'],
      ['-F perm=rwxa', 'то же, что -p, для формы -a'],
      ['-F exe=ПУТЬ', 'только когда действует эта программа'],
      ['-S вызов', 'какие системные вызовы (all — любые)'],
    ],
    recipes: [
      'Чтение секрета (файл)',
      'Чтение секретов (каталог)',
      'Изменение конфигов (каталог)',
      'Docker-креды',
      'npm-токен',
      'kubeconfig',
      'GPG-ключи (каталог)',
      'Заглушить программу для пути',
    ],
  },
}

const DATE_LOCALE = { en: 'en-GB', ru: 'ru-RU' }

function initial() {
  try {
    const v = localStorage.getItem(STORAGE_KEY)
    if (strings[v]) return v
  } catch { /* storage unavailable — use the default */ }
  return 'en'
}

let locale = $state(initial())
if (typeof document !== 'undefined') document.documentElement.lang = locale

export const i18n = {
  get locale() { return locale },
  set locale(v) {
    if (!strings[v]) return
    locale = v
    try { localStorage.setItem(STORAGE_KEY, v) } catch { /* ignore */ }
    document.documentElement.lang = v
  },
}

// t('events.more', { n: 5 }) → 'show more (5)'; missing keys fall back
// to English and finally to the key itself so nothing renders blank.
export function t(key, params) {
  let s = strings[locale][key] ?? strings.en[key] ?? key
  if (params) {
    for (const [k, v] of Object.entries(params)) s = s.replaceAll(`{${k}}`, String(v))
  }
  return s
}

export function data(key) {
  return tables[locale][key] ?? tables.en[key]
}

export function fmtTime(ts) {
  return new Date(ts).toLocaleString(DATE_LOCALE[locale], {
    day: '2-digit', month: '2-digit',
    hour: '2-digit', minute: '2-digit', second: '2-digit',
    hour12: false,
  })
}
