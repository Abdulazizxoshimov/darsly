package waitingroom

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/redis"
	"github.com/zoom/darsly/internal/infrastructure/repository"
	ws "github.com/zoom/darsly/internal/infrastructure/websocket"
	"github.com/zoom/darsly/internal/pkg/audit"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	"github.com/zoom/darsly/internal/pkg/logger"
	"github.com/zoom/darsly/internal/pkg/metrics"
	"github.com/zoom/darsly/internal/usecase/room"
	"github.com/zoom/darsly/internal/usecase/shared"
)

// Admit tokeni Redis'da shuncha vaqt saqlanadi (guest WS kechikib ulansa ham oladi).
// 15 daqiqa — past internetда uzoq reconnect oynasini qoplaydi (5 daq juda qisqa edi).
const tokenTTL = 15 * time.Minute

type useCase struct {
	repo       repository.WaitingRoomRepository
	lessonRepo repository.LessonRepository
	room       room.UseCase
	hub        *ws.Hub
	cache      redis.Cache
	log        logger.Logger
}

func New(repo repository.WaitingRoomRepository, lessonRepo repository.LessonRepository, roomUC room.UseCase, hub *ws.Hub, cache redis.Cache, log logger.Logger) UseCase {
	return &useCase{repo: repo, lessonRepo: lessonRepo, room: roomUC, hub: hub, cache: cache, log: log}
}

// validateRequestID — requestID formatini DB'ga bormasdan tekshiradi.
//
// `request_id` ochiq (auth'siz) kirish nuqtalaridan keladi — `GET /waitingroom/:id/status`
// va `GET /ws/waitingroom?request_id=`. Yaroqsiz UUID Postgres'ga yetsa 22P02 → 500
// bo'lardi (har kim auth'siz Sentry'ni to'ldirib real nosozlikni ko'rinmas qilardi).
// NotFound tanlovi sababi: shared.ValidateID izohiga qara (javob mavjud bo'lmagan
// UUID javobidan farq qilmaydi → enumeration oracle'i yo'q).
func validateRequestID(requestID string) error {
	return shared.ValidateID(requestID, "waiting room request")
}

func (uc *useCase) CreateRequest(ctx context.Context, lesson *entity.Lesson, requesterName string) (*entity.WaitingRoomRequest, error) {
	if requesterName == "" {
		requesterName = "Mehmon"
	}
	req := &entity.WaitingRoomRequest{
		ID:            uuid.NewString(),
		LessonID:      lesson.ID,
		RequesterName: requesterName,
		GuestIdentity: "guest_" + uuid.NewString(),
		Status:        entity.WaitingStatusPending,
		CreatedAt:     time.Now().UTC(),
	}
	if err := uc.repo.Create(ctx, req); err != nil {
		uc.log.Error(ctx, "waitingroom.CreateRequest: db error", logger.String("lesson_id", lesson.ID), logger.SafeString("err", err.Error()))
		return nil, err
	}

	// Mentorga real-time bildirish.
	uc.hub.Send(lesson.MentorID, ws.NewWaitingRoomRequestMsg(map[string]any{
		"request_id":     req.ID,
		"lesson_id":      lesson.ID,
		"requester_name": req.RequesterName,
		"created_at":     req.CreatedAt,
	}))

	uc.log.Info(ctx, "waitingroom: request created", logger.String("request_id", req.ID), logger.String("lesson_id", lesson.ID))
	return req, nil
}

func (uc *useCase) ListPending(ctx context.Context, mentorID, lessonID string) ([]*entity.WaitingRoomRequest, error) {
	if _, err := shared.OwnedLesson(ctx, uc.lessonRepo, mentorID, lessonID); err != nil {
		return nil, err
	}
	return uc.repo.ListPending(ctx, lessonID)
}

func (uc *useCase) Admit(ctx context.Context, mentorID, requestID string) (*entity.RoomToken, error) {
	// Authed yo'l bo'lsa-da bir xil qoida: yaroqsiz ID → 404, 500 emas.
	if err := validateRequestID(requestID); err != nil {
		return nil, err
	}
	req, err := uc.repo.GetByID(ctx, requestID)
	if err != nil {
		return nil, err
	}
	lesson, err := shared.OwnedLesson(ctx, uc.lessonRepo, mentorID, req.LessonID)
	if err != nil {
		return nil, err
	}
	rt, err := uc.admitOne(ctx, lesson, req)
	if err != nil {
		return nil, err
	}
	audit.Record(ctx, uc.log, "waitingroom.admit", mentorID,
		logger.String("request_id", req.ID), logger.String("lesson_id", req.LessonID), logger.String("guest", req.RequesterName))
	return rt, nil
}

// AdmitAll — darsning BARCHA kutayotgan so'rovlarini qabul qiladi.
//
// # Nega alohida endpoint
//
// Mobil ustozda «Hammasini kiritish» tugmasi allaqachon bor edi, lekin u
// N ta alohida `POST /waitingroom/:id/admit` chaqiruvini yuborardi: sekin
// internetda ustoz ro'yxatning yarmi kirib yarmi kirmagan holatni ko'rardi va
// har bir chaqiruv o'z rate-limit'ini yeb ketardi. Bitta amal — bitta so'rov.
//
// «Hammasi yoki hech nima» EMAS (`entity.AdmitAllResp` izohiga qara): har bir
// so'rov mustaqil, atomik claim bilan olinadi. Ro'yxat o'qilgandan keyin
// alohida admit/reject qilingan so'rov jimgina o'tkazib yuboriladi (dublikat
// token berilmaydi), qolganlari kiritilaveradi.
func (uc *useCase) AdmitAll(ctx context.Context, mentorID, lessonID string) (*entity.AdmitAllResp, error) {
	// Egalik tekshiruvi — boshqa mentor birovning darsiga hech kimni kirita olmaydi.
	// `shared.OwnedLesson` yaroqsiz UUID'ni ham DB'ga yetkazmaydi (404).
	lesson, err := shared.OwnedLesson(ctx, uc.lessonRepo, mentorID, lessonID)
	if err != nil {
		return nil, err
	}
	reqs, err := uc.repo.ListPending(ctx, lessonID)
	if err != nil {
		uc.log.Error(ctx, "waitingroom.AdmitAll: list failed",
			logger.String("lesson_id", lessonID), logger.SafeString("err", err.Error()))
		return nil, err
	}

	resp := &entity.AdmitAllResp{Total: len(reqs)}
	// KETMA-KET (parallel emas): har bir qabul LiveKit tokeni + Redis yozuvi
	// bo'lib, kutayotganlar soni odatda o'nlab — parallellik foydasi kichik,
	// lekin qisman xato holatidagi tartibsizlik va LiveKit'ga portlash yuki
	// real. `MuteAll` dan farqi shu: u yuzlab ishtirokchiga tegadi.
	for _, req := range reqs {
		if _, err := uc.admitOne(ctx, lesson, req); err != nil {
			// Bitta so'rovdagi xato qolganlarini to'xtatmaydi: 9 kishi
			// kirgani 9 kishi kirmay qolganidan yaxshi.
			uc.log.Warn(ctx, "waitingroom.AdmitAll: so'rov qabul qilinmadi",
				logger.String("request_id", req.ID), logger.SafeString("err", err.Error()))
			continue
		}
		resp.Admitted++
	}
	resp.Failed = resp.Total - resp.Admitted

	audit.Record(ctx, uc.log, "waitingroom.admit_all", mentorID,
		logger.String("lesson_id", lessonID),
		logger.Int("admitted", resp.Admitted), logger.Int("failed", resp.Failed))
	return resp, nil
}

// admitOne — bitta so'rovni atomik qabul qilib, guestga tokenni yetkazadi.
//
// `Admit` (bitta) va `AdmitAll` (ommaviy) uchun UMUMIY yadro: ikkala yo'l ham
// aynan bir xil kafolatni berishi shart — atomik claim (TOCTOU race yo'q),
// token Redis'da (WS kechiksa ham guest oladi), WS push, metrika. Avval bu
// mantiq faqat `Admit` ichida edi va ommaviy yo'l uni takrorlaganda
// nomutanosiblik paydo bo'lardi.
//
// EGALIK BU YERDA TEKSHIRILMAYDI — chaqiruvchi allaqachon tekshirgan
// (`lesson` aynan shu tekshiruv natijasi).
func (uc *useCase) admitOne(ctx context.Context, lesson *entity.Lesson, req *entity.WaitingRoomRequest) (*entity.RoomToken, error) {
	// Atomik claim: faqat pending bo'lsa admitted'ga o'tkazadi (parallel admit race oldini oladi).
	claimed, err := uc.repo.TransitionFromPending(ctx, req.ID, entity.WaitingStatusAdmitted, time.Now().UTC())
	if err != nil {
		uc.log.Error(ctx, "waitingroom.admitOne: transition failed", logger.String("request_id", req.ID), logger.SafeString("err", err.Error()))
		return nil, err
	}
	if !claimed {
		return nil, apperr.Conflict("request already decided")
	}

	rt, err := uc.room.ParticipantToken(ctx, lesson, req.GuestIdentity, req.RequesterName)
	if err != nil {
		return nil, err
	}

	// Tokenni Redis'da saqlash (WS kechiksa ham guest oladi) + real-time yuborish.
	// Cache.Set o'zi json.Marshal qiladi — struct'ni to'g'ridan-to'g'ri uzatamiz.
	_ = uc.cache.Set(ctx, tokenKey(req.ID), rt, tokenTTL)
	uc.hub.Send(req.ID, ws.NewWaitingRoomAdmittedMsg(rt))

	metrics.WaitingRoomDecisions.WithLabelValues("admit").Inc()
	return rt, nil
}

func (uc *useCase) Reject(ctx context.Context, mentorID, requestID string) error {
	if err := validateRequestID(requestID); err != nil {
		return err
	}
	req, err := uc.repo.GetByID(ctx, requestID)
	if err != nil {
		return err
	}
	if _, err := shared.OwnedLesson(ctx, uc.lessonRepo, mentorID, req.LessonID); err != nil {
		return err
	}

	claimed, err := uc.repo.TransitionFromPending(ctx, req.ID, entity.WaitingStatusRejected, time.Now().UTC())
	if err != nil {
		return err
	}
	if !claimed {
		return apperr.Conflict("request already decided")
	}
	_ = uc.cache.Del(ctx, tokenKey(req.ID))
	uc.hub.Send(req.ID, ws.NewWaitingRoomRejectedMsg(map[string]any{"request_id": req.ID}))

	metrics.WaitingRoomDecisions.WithLabelValues("reject").Inc()
	audit.Record(ctx, uc.log, "waitingroom.reject", mentorID,
		logger.String("request_id", req.ID), logger.String("lesson_id", req.LessonID))
	return nil
}

func (uc *useCase) Status(ctx context.Context, requestID string) (*entity.WaitingRoomStatusResp, error) {
	// Ochiq endpoint — DB'ga bormasdan format tekshiruvi (22P02 → 500 oldini oladi).
	if err := validateRequestID(requestID); err != nil {
		return nil, err
	}
	req, err := uc.repo.GetByID(ctx, requestID)
	if err != nil {
		return nil, err
	}
	resp := &entity.WaitingRoomStatusResp{RequestID: req.ID, Status: req.Status}
	if req.Status == entity.WaitingStatusAdmitted {
		resp.Room = uc.tokenFor(ctx, req)
	}
	return resp, nil
}

func (uc *useCase) DeliverCurrentStatus(ctx context.Context, requestID string) {
	// WS upgrade'dan keyin goroutine'da chaqiriladi — bu yerda ham DB'ga
	// yaroqsiz UUID ketmasin (log'da xato shovqini bo'lmasin).
	if validateRequestID(requestID) != nil {
		return
	}
	req, err := uc.repo.GetByID(ctx, requestID)
	if err != nil {
		return
	}
	switch req.Status {
	case entity.WaitingStatusAdmitted:
		if rt := uc.tokenFor(ctx, req); rt != nil {
			uc.hub.Send(req.ID, ws.NewWaitingRoomAdmittedMsg(rt))
		}
	case entity.WaitingStatusRejected:
		uc.hub.Send(req.ID, ws.NewWaitingRoomRejectedMsg(map[string]any{"request_id": req.ID}))
	}
}

func (uc *useCase) DeliverPendingSnapshot(ctx context.Context, mentorID string) {
	reqs, err := uc.repo.ListPendingByMentor(ctx, mentorID)
	if err != nil {
		uc.log.Warn(ctx, "waitingroom.DeliverPendingSnapshot: list failed", logger.String("mentor_id", mentorID), logger.SafeString("err", err.Error()))
		return
	}
	for _, req := range reqs {
		uc.hub.Send(mentorID, ws.NewWaitingRoomRequestMsg(map[string]any{
			"request_id":     req.ID,
			"lesson_id":      req.LessonID,
			"requester_name": req.RequesterName,
			"created_at":     req.CreatedAt,
		}))
	}
	if len(reqs) > 0 {
		uc.log.Info(ctx, "waitingroom: pending snapshot delivered", logger.String("mentor_id", mentorID), logger.Int("count", len(reqs)))
	}
}

// tokenFor — qabul qilingan (admitted) guest uchun kirish tokenini qaytaradi.
//
// BE-12: avval token FAQAT Redis'da (tokenTTL) saqlanardi. Guest mobil/brauzerni
// fon rejimiga qo'yib TTL'dan keyin qaytsa `{status:"admitted", room:null}` olardi —
// boshi berk ko'cha (joinlink qayta chaqirilsa YANGI so'rov yaratiladi va mentor
// qaytadan admit qilishi kerak bo'lardi). Endi cache bo'sh bo'lsa token DB'dagi
// GuestIdentity bilan QAYTA chiqariladi.
//
// Xavfsizlik: yangi huquq berilmaydi — `request_id` egasi allaqachon admit'da
// aynan shu tokenni olgan (capability o'zgarmaydi), identity ham o'sha
// (LiveKit uchun bir xil ishtirokchi). Dars `live` bo'lmasa token BERILMAYDI —
// tugagan/kelgusi darsga kirish yo'li ochilib qolmasin.
func (uc *useCase) tokenFor(ctx context.Context, req *entity.WaitingRoomRequest) *entity.RoomToken {
	if rt := uc.cachedToken(ctx, req.ID); rt != nil {
		return rt
	}
	l, err := uc.lessonRepo.GetByID(ctx, req.LessonID)
	if err != nil {
		uc.log.Warn(ctx, "waitingroom.tokenFor: lesson not found", logger.String("request_id", req.ID), logger.String("lesson_id", req.LessonID))
		return nil
	}
	if l.Status != entity.LessonStatusLive {
		uc.log.Info(ctx, "waitingroom.tokenFor: lesson not live, token not reissued", logger.String("request_id", req.ID), logger.String("lesson_id", l.ID))
		return nil
	}
	rt, err := uc.room.ParticipantToken(ctx, l, req.GuestIdentity, req.RequesterName)
	if err != nil {
		uc.log.Error(ctx, "waitingroom.tokenFor: reissue failed", logger.String("request_id", req.ID), logger.SafeString("err", err.Error()))
		return nil
	}
	_ = uc.cache.Set(ctx, tokenKey(req.ID), rt, tokenTTL)
	uc.log.Info(ctx, "waitingroom: token reissued after cache expiry", logger.String("request_id", req.ID), logger.String("lesson_id", l.ID))
	return rt
}

func (uc *useCase) cachedToken(ctx context.Context, requestID string) *entity.RoomToken {
	raw, err := uc.cache.Get(ctx, tokenKey(requestID))
	if err != nil || raw == "" {
		return nil
	}
	var rt entity.RoomToken
	if json.Unmarshal([]byte(raw), &rt) != nil {
		return nil
	}
	return &rt
}

func tokenKey(requestID string) string { return "waitroom:token:" + requestID }
