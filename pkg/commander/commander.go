package commander

import (
	"fmt"
	"os"
	"time"
)

type Commander struct {
	soldiers []*Soldier
}

func NewCommander(soldierCount int) *Commander {
	c := &Commander{
		soldiers: make([]*Soldier, soldierCount),
	}

	for i := 0; i < soldierCount; i++ {
		c.soldiers[i] = EnlistSoldier(i)
	}
	return c
}

func (c *Commander) Assault(targetUrl string, attackUnitRate string, attackTimeSpan string) {
	if attackUnitRate == "" {
		attackUnitRate = "0s"
	}
	attackUnitRateDuration, err := time.ParseDuration(attackUnitRate)
	if err != nil {
		fmt.Printf("Invalid attack unit rate: %s", attackUnitRate)
		os.Exit(1)
	}

	attackTimeSpanDuration, err := time.ParseDuration(attackTimeSpan)
	if err != nil {
		fmt.Printf("Invalid attack time span: %s", attackUnitRate)
		os.Exit(1)
	}

	for _, s := range c.soldiers {
		s.ClearBattleReport()
		go s.AttackUrl(AttackOrders{
			timeBetweenAttacks: attackUnitRateDuration,
			targetUrl:          targetUrl,
		})
	}

	t := time.NewTimer(attackTimeSpanDuration)
	<-t.C
	c.endAssault()
	c.summarizeAssault(attackTimeSpanDuration)
}

func (c *Commander) summarizeAssault(attackDuration time.Duration) {
	combinedReport := BattleReport{}
	for _, s := range c.soldiers {
		b := s.GiveBattleReport()
		combinedReport.ErrorCount += b.ErrorCount
		combinedReport.RequestCount += b.RequestCount
		combinedReport.ResponseTimeMsTotal += b.ResponseTimeMsTotal
	}
	fmt.Printf("%-24s%-8d\n%-24s%-8.2f\n%-24s%-8d\n%-24s%-8s\n%-24s%-8.2f", "Total Request Count:", combinedReport.RequestCount, "Average Response Time:", float64(combinedReport.ResponseTimeMsTotal)/float64(combinedReport.RequestCount), "Error Count:", combinedReport.ErrorCount, "Availability:", fmt.Sprintf("%1.2f%%", float64(combinedReport.RequestCount-combinedReport.ErrorCount)/float64(combinedReport.RequestCount)*100), "Requests Per Second:", float64(combinedReport.RequestCount)/attackDuration.Seconds())
}

func (c *Commander) endAssault() {
	for _, s := range c.soldiers {
		s.Halt()
	}
}
