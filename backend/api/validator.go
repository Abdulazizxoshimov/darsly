package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"sync"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	vpkg "github.com/zoom/darsly/internal/pkg/validator"
)

// structValidator gin'ning binding.StructValidator interfeysini `validate` teglari
// asosida amalga oshiradi. gin default'i "binding" tegini o'qiydi — bizning entity'lar
// esa "validate" tegidan foydalanadi, shuning uchun ShouldBindJSON validatsiyani
// o'tkazib yuborardi. Bu adapter shuni tuzatadi.
type structValidator struct {
	once sync.Once
	v    *validator.Validate
}

func (s *structValidator) lazy() {
	s.once.Do(func() {
		s.v = vpkg.Engine()
		registerCustomValidations(s.v)
	})
}

// registerCustomValidations — loyihaning maxsus `validate` teglari.
//
// httpurl: faqat http/https sxemali, hostli URL. go-playground `url` tegi HAR
// QANDAY sxemani (javascript:, data:, file:) qabul qiladi — avatar_url kabi
// maydonlar frontend'da <img src>/<a href> ga tushsa stored-XSS vektori bo'ladi.
func registerCustomValidations(v *validator.Validate) {
	_ = v.RegisterValidation("httpurl", func(fl validator.FieldLevel) bool {
		u, err := url.Parse(fl.Field().String())
		if err != nil || u.Host == "" {
			return false
		}
		return u.Scheme == "http" || u.Scheme == "https"
	})
}

func (s *structValidator) ValidateStruct(obj any) error {
	if obj == nil {
		return nil
	}
	value := reflect.ValueOf(obj)
	switch value.Kind() {
	case reflect.Ptr:
		if value.IsNil() {
			return nil
		}
		return s.ValidateStruct(value.Elem().Interface())
	case reflect.Struct:
		s.lazy()
		// vpkg.Validate — xom `validator.ValidationErrors` o'rniga foydalanuvchiga
		// xavfsiz *vpkg.ValidationError qaytaradi. Xom xato Error() matni Go ichki
		// tuzilmasini oshkor qilardi:
		//   "Key: 'RegisterReq.password' Error:Field validation for 'password' failed on the 'min' tag"
		// endi: "password: must be at least 10 characters" (struct nomi va tag nomi yo'q).
		return vpkg.Validate(obj)
	case reflect.Slice, reflect.Array:
		s.lazy()
		for i := 0; i < value.Len(); i++ {
			if err := s.ValidateStruct(value.Index(i).Interface()); err != nil {
				return err
			}
		}
		return nil
	default:
		return nil
	}
}

func (s *structValidator) Engine() any {
	s.lazy()
	return s.v
}

// ─── JSON binding xatolarini tozalash ───────────────────────────────────────
//
// gin'ning JSON binder'i encoding/json xatosini o'zgartirmasdan qaytaradi, handler esa
// uni `hs.BadRequest(c, err.Error())` bilan klientga uzatadi. Natijada Go ichki
// tuzilmasi oshkor bo'lardi:
//
//	"json: cannot unmarshal number into Go struct field LoginReq.email of type string"
//
// Bu kuchsiz info-leak (struct nomlari + maydon yo'llari hujumchiga API ichki
// modelini chizib beradi). Quyidagi o'ram xatoni JSON maydon nomi darajasida
// tushunarli, lekin ichki tafsilotsiz matnga aylantiradi. Handler'larni (18 ta
// chaqiruv joyi) o'zgartirish kerak emas — tuzatish manbada.
type safeJSONBinding struct{ inner binding.BindingBody }

func (b safeJSONBinding) Name() string { return b.inner.Name() }

func (b safeJSONBinding) Bind(req *http.Request, obj any) error {
	return sanitizeBindError(b.inner.Bind(req, obj))
}

func (b safeJSONBinding) BindBody(body []byte, obj any) error {
	return sanitizeBindError(b.inner.BindBody(body, obj))
}

// sanitizeBindError binding/validatsiya xatosini klientga xavfsiz shaklga keltiradi.
// Ma'no saqlanadi (qaysi maydon, nima kutilgan), ichki tafsilot chiqmaydi.
func sanitizeBindError(err error) error {
	if err == nil {
		return nil
	}

	// Validatsiya xatosi allaqachon xavfsiz va foydali ("password: must be at least
	// 10 characters") — o'zgartirmaymiz.
	var ve *vpkg.ValidationError
	if errors.As(err, &ve) {
		return err
	}

	// Tur mos kelmadi: JSON maydon nomini ko'rsatamiz (xavfsiz), lekin Go struct
	// nomini ("LoginReq") va Go tur nomini emas.
	var ute *json.UnmarshalTypeError
	if errors.As(err, &ute) {
		if ute.Field != "" {
			return fmt.Errorf("%s: expected %s", ute.Field, jsonTypeName(ute.Type))
		}
		return errors.New("invalid value type in request body")
	}

	// Buzuq/chala JSON.
	var se *json.SyntaxError
	if errors.As(err, &se) {
		return errors.New("malformed JSON body")
	}
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return errors.New("malformed JSON body")
	}
	if errors.Is(err, io.EOF) {
		return errors.New("request body is required")
	}

	// Qolgan hamma holat: umumiy matn (hech narsa oshkor qilinmaydi).
	return errors.New("invalid request body")
}

// jsonTypeName Go turini JSON tushunchasiga o'giradi — `entity.X` kabi ichki tur
// nomlari klientga chiqmasin.
func jsonTypeName(t reflect.Type) string {
	if t == nil {
		return "another type"
	}
	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Slice, reflect.Array:
		return "array"
	case reflect.Map, reflect.Struct:
		return "object"
	default:
		return "another type"
	}
}

// installValidator gin'ning global validator'ini `validate` teglarini yurituvchi
// adapter bilan, JSON binder'ini esa xato-tozalovchi o'ram bilan almashtiradi.
// NewRouter boshida bir marta chaqiriladi.
func installValidator() {
	binding.Validator = &structValidator{}
	// Ikki marta o'ralib qolmasin (NewRouter testlarda bir necha marta chaqiriladi).
	if _, already := binding.JSON.(safeJSONBinding); !already {
		binding.JSON = safeJSONBinding{inner: binding.JSON}
	}
}
