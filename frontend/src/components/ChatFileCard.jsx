import { useState } from 'react'
import { Download, FileSpreadsheet, FileText, FileType, ImageIcon } from 'lucide-react'
import { formatBytes } from '../lib/format'
import { safeUrl } from '../lib/url'

// Chatga biriktirilgan fayl kartochkasi (jonli xonada ham, tarix modalida ham).
//
// `file.url` PRESIGNED va 1 soatlik: server uni har javobda qayta imzolaydi va
// bazada saqlamaydi. Shuning uchun havola KESHLANMAYDI va yangi tabda ochiladi —
// yuklab olishni brauzerning o'ziga qoldiramiz (mobil brauzerlar `download`
// atributini cross-origin havolada baribir e'tiborsiz qoldiradi).

// Element QAYTARADI, komponent emas: komponentni render paytida yasash
// (`const Icon = pick()` → `<Icon/>`) React'ning identitet qoidasini buzadi va
// lint uni to'g'ri to'sadi — har renderda yangi tur daraxtni qayta yaratardi.
function fileIcon(mime = '', name = '') {
  if (mime === 'application/pdf' || name.toLowerCase().endsWith('.pdf')) return <FileType size={20} />
  if (/sheet|excel|csv/i.test(mime) || /\.(xlsx?|csv)$/i.test(name)) return <FileSpreadsheet size={20} />
  if (mime.startsWith('image/')) return <ImageIcon size={20} />
  return <FileText size={20} />
}

export function ChatFileCard({ file }) {
  // Presigned havola 1 soatlik: uzoq darsda eski xabarning rasmi ochilmay
  // qolishi mumkin. Bunda brauzerning "buzilgan rasm" belgisi o'rniga oddiy
  // ikonkaga tushamiz — kartochka nom va hajm bilan o'qilishicha qoladi.
  const [thumbBroken, setThumbBroken] = useState(false)
  if (!file) return null
  // Faqat http(s)/blob havola `href`/`src` ga tushadi (`lib/url`): boshqa
  // sxema bo'lsa kartochka bosilmaydigan bo'lib qoladi, lekin nom/hajm ko'rinadi.
  const url = safeUrl(file.url)
  const isImage = !!url && String(file.mime || '').startsWith('image/') && !thumbBroken

  return (
    <a
      className="chat-file"
      href={url || undefined}
      target="_blank"
      rel="noopener noreferrer"
      title={`${file.name} — ochish`}
    >
      {isImage ? (
        // Kichik ko'rish. Rasm ochilmasa (havola muddati tugagan bo'lsa)
        // kartochka baribir nom + hajm bilan qoladi.
        <img
          className="chat-file__thumb"
          src={url}
          alt={file.name}
          loading="lazy"
          onError={() => setThumbBroken(true)}
        />
      ) : (
        <span className="chat-file__icon">{fileIcon(file.mime, file.name)}</span>
      )}
      <span className="chat-file__meta">
        <span className="chat-file__name truncate">{file.name}</span>
        <span className="chat-file__size">{formatBytes(file.size)}</span>
      </span>
      <span className="chat-file__action" aria-hidden="true">
        <Download size={16} />
      </span>
    </a>
  )
}
