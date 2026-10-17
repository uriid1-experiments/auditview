// Draggable table column borders (Wireshark style): only the border
// between two adjacent columns moves.
export function createColResize({ defaults, order, storageKey, min = 60 }) {
  let colW = $state(load())

  function load() {
    try {
      const saved = JSON.parse(localStorage.getItem(storageKey))
      if (saved && typeof saved === 'object') return { ...defaults, ...saved }
    } catch { /* corrupt localStorage — fall back to defaults */ }
    return { ...defaults }
  }

  function startDrag(e, key) {
    e.preventDefault()
    const next = order[order.indexOf(key) + 1]
    if (!next) return
    const th = e.currentTarget.closest('th')
    const thNext = th.nextElementSibling
    const startX = e.clientX
    const startA = th.offsetWidth
    const startB = thNext.offsetWidth
    const move = (ev) => {
      let dx = ev.clientX - startX
      dx = Math.max(min - startA, Math.min(dx, startB - min))
      colW[key] = startA + dx
      colW[next] = startB - dx
    }
    const up = () => {
      window.removeEventListener('pointermove', move)
      window.removeEventListener('pointerup', up)
      localStorage.setItem(storageKey, JSON.stringify(colW))
    }
    window.addEventListener('pointermove', move)
    window.addEventListener('pointerup', up)
  }

  function reset() {
    colW = { ...defaults }
    localStorage.removeItem(storageKey)
  }

  return {
    get w() { return colW },
    order,
    startDrag,
    reset,
  }
}
