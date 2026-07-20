import DOMPurify from 'dompurify'
import { marked } from 'marked'

const forbiddenTags = [
  'script', 'style', 'iframe', 'form', 'input', 'button', 'textarea', 'select',
  'option', 'object', 'embed', 'link', 'meta',
]

export function renderBrandPanelContent(source: string): string {
  if (!source.trim()) return ''
  const html = marked.parse(source, { async: false, breaks: true, gfm: true })
  return DOMPurify.sanitize(html, {
    FORBID_TAGS: forbiddenTags,
    FORBID_ATTR: ['style', 'srcdoc'],
  })
}
