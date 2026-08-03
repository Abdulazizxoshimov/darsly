-- Yozuv faylining BOSHIDAN kesilgan soniyalar (transkoddagi "o'lik" qism).
--
-- Nega kerak: arxiv sahifasida chat xabari bosilganda video o'sha lahzaga
-- sakraydi. Sakrash `offset_sec` bo'yicha hisoblanadi va uning nol nuqtasi
-- videoning t=0 lahzasi bo'lishi SHART. Transkod yozuv boshidagi qora va jim
-- qismni kesgach video t=0 oldinga suriladi — bu ustun aynan o'sha siljishni
-- saqlaydi. Busiz butun chat ~10-15 soniyaga siljib ketardi.
--
-- 0 — kesilmagan (yoki eski qatorlar): xulq avvalgidek qoladi.
ALTER TABLE recordings
    ADD COLUMN IF NOT EXISTS content_offset_sec INTEGER NOT NULL DEFAULT 0;
