import { describe, expect, it } from 'vitest'
import { isSafeUrl, safeUrl } from './url'

// Serverdan kelgan `url` `<a href>` ga tekshiruvsiz tushardi — `javascript:`
// sxemasi bosilganda kod bajarilardi.
describe('isSafeUrl / safeUrl', () => {
  it('http(s) va blob o‘tadi', () => {
    expect(isSafeUrl('https://minio/x.mp4?X-Amz=1')).toBe(true)
    expect(isSafeUrl('http://localhost:9020/f.pdf')).toBe(true)
    expect(isSafeUrl(' HTTPS://x/y ')).toBe(true)
    expect(isSafeUrl('blob:mock/uy_ishi.pdf')).toBe(true)
  })

  it('javascript: / data: / nisbiy / bo‘sh → xavfsiz EMAS', () => {
    expect(isSafeUrl('javascript:alert(1)')).toBe(false)
    expect(isSafeUrl('JavaScript:alert(1)')).toBe(false)
    expect(isSafeUrl('data:text/html,<script>')).toBe(false)
    expect(isSafeUrl('/relative')).toBe(false)
    expect(isSafeUrl('')).toBe(false)
    expect(isSafeUrl(null)).toBe(false)
    expect(isSafeUrl(42)).toBe(false)
  })

  it('safeUrl xavfsiz bo‘lsa o‘zini, aks holda null qaytaradi', () => {
    expect(safeUrl('https://x')).toBe('https://x')
    expect(safeUrl('javascript:1')).toBeNull()
  })
})
