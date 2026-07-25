// Перевод правил auditd в человеческий язык и обратно.

export const PERMS = [
  ['r', 'чтение'],
  ['w', 'изменение'],
  ['x', 'запуск'],
  ['a', 'атрибуты'],
]
const PERM_MAP = Object.fromEntries(PERMS)

export function permWords(perms) {
  return [...(perms || '')].map((c) => PERM_MAP[c] ?? c).join(', ')
}

export function baseName(p) {
  const parts = (p || '').split('/')
  return parts[parts.length - 1] || p
}

// Правило можно редактировать формой, если мы понимаем все его части.
export function editable(e) {
  if (e.kind !== 'watch' && e.kind !== 'syscall') return false
  if (e.err || !e.path) return false
  const extra = Object.keys(e.fields ?? {}).filter(
    (k) => k !== 'exe' && !(k === 'syscall' && e.fields[k] === 'all')
  )
  return extra.length === 0
}

// Человеческая фраза для карточки. null → показывать сырую строку.
export function describe(e) {
  if (!editable(e)) return null
  const scope = e.path_kind === 'dir' ? ' (весь каталог)' : ''
  if (e.action === 'never') {
    const who = e.fields?.exe ? baseName(e.fields.exe) : 'любая программа'
    const acts = e.perms ? permWords(e.perms) : 'любое обращение'
    return {
      icon: '🔇',
      text: `Без событий: ${who} — ${acts}: ${e.path}${scope}`,
    }
  }
  const acts = permWords(e.perms || 'rwxa')
  return {
    icon: '👁',
    text: `Следить: ${acts} — ${e.path}${scope}`,
    key: e.key || '',
  }
}

// Значения формы из распарсенного правила.
export function formFrom(e) {
  return {
    action: e.action === 'never' ? 'never' : 'always',
    scope: e.path_kind === 'dir' ? 'dir' : 'exact',
    path: e.path ?? '',
    perms: e.perms ?? (e.action === 'never' ? '' : 'r'),
    key: e.key ?? '',
    exe: e.fields?.exe ?? '',
  }
}

// Строка правила из значений формы.
export function generate(f) {
  const path = f.path.trim()
  if (!path) return ''
  if (f.action === 'never') {
    let l = '-a never,exit'
    if (f.exe.trim()) l += ` -F exe=${f.exe.trim()}`
    l += ` -F ${f.scope === 'dir' ? 'dir' : 'path'}=${path}`
    if (f.perms) l += ` -F perm=${f.perms}`
    return l
  }
  if (f.scope === 'dir') {
    let l = `-a always,exit -F dir=${path}`
    if (f.perms) l += ` -F perm=${f.perms}`
    if (f.key.trim()) l += ` -k ${f.key.trim()}`
    return l
  }
  let l = `-w ${path}`
  if (f.perms) l += ` -p ${f.perms}`
  if (f.key.trim()) l += ` -k ${f.key.trim()}`
  return l
}

// Шпаргалка по флагам для режима эксперта.
export const FLAG_LEGEND = [
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
]

// Готовые заготовки для конструктора.
export const RECIPES = [
  { label: 'Чтение секрета (файл)', f: { action: 'always', scope: 'exact', path: '', perms: 'r', key: 'cred_', exe: '' } },
  { label: 'Чтение секретов (каталог)', f: { action: 'always', scope: 'dir', path: '', perms: 'r', key: 'cred_', exe: '' } },
  { label: 'Изменение конфигов (каталог)', f: { action: 'always', scope: 'dir', path: '', perms: 'wa', key: 'persist_', exe: '' } },
  { label: 'Docker-креды', f: { action: 'always', scope: 'exact', path: '/home/uriid1/.docker/config.json', perms: 'r', key: 'cred_docker', exe: '' } },
  { label: 'npm-токен', f: { action: 'always', scope: 'exact', path: '/home/uriid1/.npmrc', perms: 'r', key: 'cred_npm', exe: '' } },
  { label: 'kubeconfig', f: { action: 'always', scope: 'exact', path: '/home/uriid1/.kube/config', perms: 'r', key: 'cred_kube', exe: '' } },
  { label: 'GPG-ключи (каталог)', f: { action: 'always', scope: 'dir', path: '/home/uriid1/.gnupg', perms: 'r', key: 'cred_gpg', exe: '' } },
  { label: 'Заглушить программу для пути', f: { action: 'never', scope: 'exact', path: '', perms: 'r', key: '', exe: '' } },
]
