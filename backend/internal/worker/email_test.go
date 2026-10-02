package worker

import (
	"testing"

	"github.com/zoom/darsly/internal/infrastructure/email"
)

// Poison-loop: Attempt oshadi va chegarada to'xtaydi.
func TestNextAttempt_StopsAtLimit(t *testing.T) {
	job := email.EmailJob{Attempt: 1}
	tries := 1
	for {
		retry, next := nextAttempt(job, maxEmailAttempts)
		if !retry {
			break
		}
		if next.Attempt != job.Attempt+1 {
			t.Fatalf("Attempt oshmadi: %d -> %d", job.Attempt, next.Attempt)
		}
		job = next
		tries++
		if tries > 100 {
			t.Fatal("cheksiz loop")
		}
	}
	if tries != maxEmailAttempts {
		t.Fatalf("urinishlar soni %d, kutilgan %d", tries, maxEmailAttempts)
	}
}

func TestNextAttempt_ZeroAttemptTreatedAsFirst(t *testing.T) {
	retry, next := nextAttempt(email.EmailJob{}, 3)
	if !retry || next.Attempt != 2 {
		t.Fatalf("retry=%v attempt=%d", retry, next.Attempt)
	}
}
