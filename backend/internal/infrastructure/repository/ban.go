package repository

import "context"

// BanRepository — dars-darajali ban'ning DURABLE (PG) reyestri (audit R2).
//
// Redis (`room:ban:*`) tezkor kesh bo'lib qoladi; bu esa Redis o'chsa ham
// kick amalda qolishi uchun zaxira. Faqat asosiy gate (token berish +
// `room.EnforceJoin` webhook'i) o'qiydi — ochiq amal guardlari Redis-only.
type BanRepository interface {
	// AddBan — ishtirokchini dars ban ro'yxatiga qo'shadi (upsert; takror kick
	// no-op). displayName ma'lum bo'lsa audit uchun saqlanadi.
	AddBan(ctx context.Context, lessonID, identity, displayName string) error
	// IsBanned — ishtirokchi shu darsdan chiqarilganmi (durable manba).
	IsBanned(ctx context.Context, lessonID, identity string) (bool, error)
}
