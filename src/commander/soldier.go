package commander

import (
	"fmt"
	"net/http"
	"time"
)

type Soldier struct {
	id          int
	report      *BattleReport
	isAttacking bool
}

func EnlistSoldier(id int) *Soldier {
	return &Soldier{
		id:     id,
		report: &BattleReport{},
	}
}

func (s *Soldier) GetId() int {
	return s.id
}

func (s *Soldier) AttackUrl(o AttackOrders) {
	s.isAttacking = true
	for {
		if !s.isAttacking {
			return
		}
		b := time.Now()
		r, err := http.DefaultClient.Get(o.targetUrl)
		a := time.Now()
		if err != nil || r.StatusCode >= 400 {
			s.report.ErrorCount++
			if err != nil {
				fmt.Println(err.Error())
			}
		}
		s.report.ResponseTimeMsTotal += int(a.Sub(b)) / 1000000
		s.report.RequestCount++
		if s.isAttacking && o.timeBetweenAttacks != 0 {
			time.Sleep(o.timeBetweenAttacks)
		}
	}
}

func (s *Soldier) Halt() {
	s.isAttacking = false
}

func (s *Soldier) ClearBattleReport() {
	s.report = &BattleReport{}
}

func (s *Soldier) GiveBattleReport() BattleReport {
	return *s.report
}
