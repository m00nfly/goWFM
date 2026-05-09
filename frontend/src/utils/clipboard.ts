/**
 * 将文本复制到剪贴板。
 * 优先使用异步 Clipboard API（仅在 HTTPS 或 localhost 等安全上下文下可用），
 * 否则降级为 document.execCommand('copy')，以兼容通过 HTTP + IP 访问的场景。
 */
export async function copyToClipboard(text: string): Promise<boolean> {
  // 优先使用现代 Clipboard API
  if (typeof navigator !== 'undefined' && navigator.clipboard && window.isSecureContext) {
    try {
      await navigator.clipboard.writeText(text)
      return true
    } catch (_) {
      // 某些浏览器/权限下可能抛错，继续走降级方案
    }
  }

  // 降级：使用隐藏的 textarea + execCommand('copy')
  try {
    const textarea = document.createElement('textarea')
    textarea.value = text
    textarea.setAttribute('readonly', '')
    textarea.style.position = 'fixed'
    textarea.style.top = '0'
    textarea.style.left = '0'
    textarea.style.width = '1px'
    textarea.style.height = '1px'
    textarea.style.padding = '0'
    textarea.style.border = 'none'
    textarea.style.outline = 'none'
    textarea.style.boxShadow = 'none'
    textarea.style.background = 'transparent'
    textarea.style.opacity = '0'
    document.body.appendChild(textarea)

    const selection = document.getSelection()
    const previousRange = selection && selection.rangeCount > 0 ? selection.getRangeAt(0) : null

    textarea.focus()
    textarea.select()
    textarea.setSelectionRange(0, textarea.value.length)

    const ok = document.execCommand('copy')

    document.body.removeChild(textarea)

    // 还原之前的选区
    if (previousRange && selection) {
      selection.removeAllRanges()
      selection.addRange(previousRange)
    }

    return ok
  } catch (_) {
    return false
  }
}
