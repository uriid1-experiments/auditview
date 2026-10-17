// Translating auditd rules into plain language and back.
import { t, data } from './i18n.svelte.js'

export const PERM_CODES = ['r', 'w', 'x', 'a']

// [code, label] pairs in the current UI language.
export function perms() {
  return PERM_CODES.map((c) => [c, t('perm.' + c)])
}

export function permWords(p) {
  return [...(p || '')].map((c) => (PERM_CODES.includes(c) ? t('perm.' + c) : c)).join(', ')
}

export function baseName(p) {
  const parts = (p || '').split('/')
  return parts[parts.length - 1] || p
}

// A rule can be edited with the form only if we understand all of its parts.
export function editable(e) {
  if (e.kind !== 'watch' && e.kind !== 'syscall') return false
  if (e.err || !e.path) return false
  const extra = Object.keys(e.fields ?? {}).filter(
    (k) => k !== 'exe' && !(k === 'syscall' && e.fields[k] === 'all')
  )
  return extra.length === 0
}

// Human phrase for the card. null → show the raw line.
export function describe(e) {
  if (!editable(e)) return null
  const scope = e.path_kind === 'dir' ? t('lang.wholeDir') : ''
  if (e.action === 'never') {
    const who = e.fields?.exe ? baseName(e.fields.exe) : t('lang.anyProgram')
    const acts = e.perms ? permWords(e.perms) : t('lang.anyAccess')
    return {
      icon: '🔇',
      text: t('lang.noEvents', { who, acts, path: e.path, scope }),
    }
  }
  const acts = permWords(e.perms || 'rwxa')
  return {
    icon: '👁',
    text: t('lang.watch', { acts, path: e.path, scope }),
    key: e.key || '',
  }
}

// Form values from a parsed rule.
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

// Rule line from form values.
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

// Flag cheat sheet for expert mode, in the current language.
export function flagLegend() {
  return data('legend')
}

// Ready-made templates for the rule builder. auditd needs absolute paths,
// so home-relative recipes carry a USER placeholder to replace.
const RECIPE_FORMS = [
  { action: 'always', scope: 'exact', path: '', perms: 'r', key: 'cred_', exe: '' },
  { action: 'always', scope: 'dir', path: '', perms: 'r', key: 'cred_', exe: '' },
  { action: 'always', scope: 'dir', path: '', perms: 'wa', key: 'persist_', exe: '' },
  { action: 'always', scope: 'exact', path: '/home/USER/.docker/config.json', perms: 'r', key: 'cred_docker', exe: '' },
  { action: 'always', scope: 'exact', path: '/home/USER/.npmrc', perms: 'r', key: 'cred_npm', exe: '' },
  { action: 'always', scope: 'exact', path: '/home/USER/.kube/config', perms: 'r', key: 'cred_kube', exe: '' },
  { action: 'always', scope: 'dir', path: '/home/USER/.gnupg', perms: 'r', key: 'cred_gpg', exe: '' },
  { action: 'never', scope: 'exact', path: '', perms: 'r', key: '', exe: '' },
]

export function recipes() {
  const labels = data('recipes')
  return RECIPE_FORMS.map((f, i) => ({ label: labels[i], f }))
}
