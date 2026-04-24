import { describe, it, expect } from 'vitest'
import { render } from '@testing-library/react'
import { linkify } from '@/utils/linkify'

function renderNodes(body: string) {
  const { container } = render(<div>{linkify(body)}</div>)
  return container.firstChild as HTMLDivElement
}

describe('linkify', () => {
  it('returns plain string for body without URLs', () => {
    const nodes = linkify('hello world')
    expect(nodes).toHaveLength(1)
    expect(nodes[0]).toBe('hello world')
  })

  it('renders a URL as an anchor with safe attrs', () => {
    const div = renderNodes('visit https://example.com now')
    const a = div.querySelector('a')
    expect(a).not.toBeNull()
    expect(a?.getAttribute('href')).toBe('https://example.com')
    expect(a?.getAttribute('target')).toBe('_blank')
    expect(a?.getAttribute('rel')).toBe('noopener noreferrer')
    expect(a?.textContent).toBe('https://example.com')
  })

  it('excludes trailing punctuation from the URL', () => {
    const div = renderNodes('see https://example.com.')
    const a = div.querySelector('a')
    expect(a?.getAttribute('href')).toBe('https://example.com')
    // The trailing period remains as plain text.
    expect(div.textContent).toBe('see https://example.com.')
  })

  it('excludes trailing paren', () => {
    const div = renderNodes('(see https://example.com)')
    const a = div.querySelector('a')
    expect(a?.getAttribute('href')).toBe('https://example.com')
    expect(div.textContent).toBe('(see https://example.com)')
  })

  it('handles multiple URLs in one body', () => {
    const div = renderNodes('one https://a.example two https://b.example done')
    const anchors = div.querySelectorAll('a')
    expect(anchors.length).toBe(2)
    expect(anchors[0].getAttribute('href')).toBe('https://a.example')
    expect(anchors[1].getAttribute('href')).toBe('https://b.example')
  })

  it('escapes HTML-like content as plain text (no script execution)', () => {
    const div = renderNodes('<script>alert("xss")</script>')
    expect(div.querySelector('script')).toBeNull()
    expect(div.textContent).toBe('<script>alert("xss")</script>')
  })

  it('supports http as well as https', () => {
    const div = renderNodes('insecure http://example.com')
    const a = div.querySelector('a')
    expect(a?.getAttribute('href')).toBe('http://example.com')
  })

  it('is idempotent across repeated calls (no regex state leak)', () => {
    const first = linkify('https://example.com').length
    const second = linkify('https://example.com').length
    expect(first).toBe(second)
  })
})
