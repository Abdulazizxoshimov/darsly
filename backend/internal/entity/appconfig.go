package entity

// AppConfig — ochiq `GET /api/v1/app-config` javobi.
//
// Mobil ilova (side-load APK) ishga tushganda shu endpointni so'raydi: agar o'z
// versiyasi `min_version`dan past bo'lsa — ishlashdan bosh tortadi va `apk_url`ni
// ko'rsatadi. Store'siz tarqatilgan APK'da avtomatik yangilanish yo'q, shuning uchun
// buzuq versiyani to'xtatishning yagona yo'li — server tomonidagi bu "kill switch".
type AppConfig struct {
	Android AppPlatformConfig `json:"android"`
	// AllowOpenRegistration — ochiq ro'yxatdan o'tish yoqiqmi. Web login sahifasi
	// shu bayroqqa qarab "Ro'yxatdan o'tish" bo'limini ko'rsatadi yoki yashiradi
	// (server baribir o'zi ham tekshiradi — bu faqat UI uchun signal).
	AllowOpenRegistration bool `json:"allow_open_registration"`
}

// AppPlatformConfig — bitta platforma uchun versiya siyosati.
type AppPlatformConfig struct {
	// MinVersion — shu versiyadan past klient ishlamasligi kerak (semver).
	MinVersion string `json:"min_version"`
	// LatestVersion — mavjud eng yangi versiya (yumshoq taklif uchun).
	LatestVersion string `json:"latest_version"`
	// APKURL — yuklab olish havolasi.
	APKURL string `json:"apk_url"`
	// ForceUpdate — true bo'lsa klient versiyasidan qat'i nazar yangilashga majburlanadi
	// (favqulodda to'xtatish).
	ForceUpdate bool `json:"force_update"`
	// ReleaseNotes — foydalanuvchiga ko'rsatiladigan o'zgarishlar matni.
	ReleaseNotes string `json:"release_notes"`
}
