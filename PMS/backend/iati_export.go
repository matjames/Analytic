package main

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ─── IATI 2.03 XML Data Structures ──────────────────────────────────────────

type IATINarrative struct {
	XMLName xml.Name `xml:"narrative"`
	Lang    string   `xml:"xml:lang,attr,omitempty"`
	Value   string   `xml:",chardata"`
}

type IATITitle struct {
	XMLName   xml.Name        `xml:"title"`
	Narrative []IATINarrative `xml:"narrative"`
}

type IATIDescription struct {
	XMLName   xml.Name        `xml:"description"`
	Type      string          `xml:"type,attr,omitempty"`
	Narrative []IATINarrative `xml:"narrative"`
}

type IATIReportingOrg struct {
	XMLName   xml.Name        `xml:"reporting-org"`
	Ref       string          `xml:"ref,attr"`
	Type      string          `xml:"type,attr"`
	Narrative []IATINarrative `xml:"narrative"`
}

type IATIActivityDate struct {
	XMLName xml.Name `xml:"activity-date"`
	Type    string   `xml:"type,attr"` // 1=planned start, 2=actual start, 3=planned end, 4=actual end
	ISODate string   `xml:"iso-date,attr"`
}

type IATISector struct {
	XMLName    xml.Name `xml:"sector"`
	Vocabulary string   `xml:"vocabulary,attr,omitempty"`
	Code       string   `xml:"code,attr"`
}

type IATIRecipientCountry struct {
	XMLName   xml.Name        `xml:"recipient-country"`
	Code      string          `xml:"code,attr"`
	Narrative []IATINarrative `xml:"narrative"`
}

type IATIBudgetValue struct {
	XMLName   xml.Name `xml:"value"`
	Currency  string   `xml:"currency,attr"`
	ValueDate string   `xml:"value-date,attr"`
	Amount    float64  `xml:",chardata"`
}

type IATIBudget struct {
	XMLName     xml.Name        `xml:"budget"`
	Type        string          `xml:"type,attr,omitempty"`
	Status      string          `xml:"status,attr,omitempty"`
	PeriodStart string          `xml:"period-start/@iso-date,omitempty"`
	PeriodEnd   string          `xml:"period-end/@iso-date,omitempty"`
	Value       IATIBudgetValue `xml:"value"`
}

type IATIActivity struct {
	XMLName         xml.Name         `xml:"iati-activity"`
	Version         string           `xml:"version,attr"`
	DefaultCurrency string           `xml:"default-currency,attr"`
	IATIIdentifier  string           `xml:"iati-identifier"`
	ReportingOrg    IATIReportingOrg `xml:"reporting-org"`
	Title           IATITitle        `xml:"title"`
	Description     IATIDescription  `xml:"description"`
	ActivityStatus  struct {
		Code string `xml:"code,attr"`
	} `xml:"activity-status"`
	ActivityDates    []IATIActivityDate   `xml:"activity-date"`
	RecipientCountry IATIRecipientCountry `xml:"recipient-country"`
	Sector           IATISector           `xml:"sector"`
	Budget           *IATIBudget          `xml:"budget,omitempty"`
}

type IATIActivities struct {
	XMLName           xml.Name       `xml:"iati-activities"`
	Version           string         `xml:"version,attr"`
	GeneratedDatetime string         `xml:"generated-datetime,attr"`
	Activities        []IATIActivity `xml:"iati-activity"`
}

// dbExportIATIActivities exports all workspace projects as standard IATI 2.03 XML.
func dbExportIATIActivities(c *gin.Context) {
	if DB == nil {
		c.Header("Content-Type", "application/xml; charset=utf-8")
		c.String(http.StatusOK, xml.Header+`<iati-activities version="2.03" generated-datetime="`+time.Now().UTC().Format(time.RFC3339)+`"/>`)
		return
	}

	rows, err := DB.Query(`
		SELECT id, code, name, description, stage, progress, org, portfolio, programme,
		       owner, start_date, end_date, target_geo, tags, budget_total, spent_total,
		       risks_count, issues_count, created_time, updated_time, COALESCE(workspace_id, '')
		FROM pms.projects
		WHERE (workspace_id = NULLIF($1, '') OR NULLIF($1, '') IS NULL)
		ORDER BY created_time DESC
	`, workspaceIDContext(c))
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load projects: "+err.Error())
		return
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var p Project
		err := rows.Scan(&p.ID, &p.Code, &p.Name, &p.Description, &p.Stage, &p.Progress,
			&p.Org, &p.Portfolio, &p.Programme, &p.Owner, &p.StartDate, &p.EndDate,
			&p.TargetGeo, &p.Tags, &p.BudgetTotal, &p.SpentTotal, &p.RisksCount, &p.IssuesCount, &p.CreatedTime, &p.UpdatedTime, &p.WorkspaceID)
		if err == nil {
			projects = append(projects, p)
		}
	}

	activities := make([]IATIActivity, 0, len(projects))
	for _, p := range projects {
		statusCode := "2" // 1=Pipeline, 2=Implementation, 3=Finalisation, 4=Closed, 5=Cancelled
		switch p.Stage {
		case "planning", "draft", "initiation":
			statusCode = "1"
		case "in_progress", "active", "execution":
			statusCode = "2"
		case "completed", "closed":
			statusCode = "4"
		case "cancelled", "suspended":
			statusCode = "5"
		}

		dates := make([]IATIActivityDate, 0)
		if p.StartDate != "" {
			dates = append(dates, IATIActivityDate{Type: "1", ISODate: p.StartDate})
		}
		if p.EndDate != "" {
			dates = append(dates, IATIActivityDate{Type: "3", ISODate: p.EndDate})
		}

		act := IATIActivity{
			Version:         "2.03",
			DefaultCurrency: "USD",
			IATIIdentifier:  fmt.Sprintf("UG-GOV-STATGATE-%s", p.ID),
			ReportingOrg: IATIReportingOrg{
				Ref:  "UG-GOV-STATGATE",
				Type: "10", // Government
				Narrative: []IATINarrative{
					{Lang: "en", Value: "StatGate National Development Monitoring Authority"},
				},
			},
			Title: IATITitle{
				Narrative: []IATINarrative{{Lang: "en", Value: p.Name}},
			},
			Description: IATIDescription{
				Type:      "1", // General
				Narrative: []IATINarrative{{Lang: "en", Value: p.Description}},
			},
			RecipientCountry: IATIRecipientCountry{
				Code:      "UG",
				Narrative: []IATINarrative{{Lang: "en", Value: "Uganda"}},
			},
			Sector: IATISector{
				Vocabulary: "1",     // OECD DAC CRS
				Code:       "15110", // Public sector policy and administrative management
			},
		}
		act.ActivityStatus.Code = statusCode

		if p.BudgetTotal > 0 {
			act.Budget = &IATIBudget{
				Type:   "1",
				Status: "2",
				Value: IATIBudgetValue{
					Currency:  "USD",
					ValueDate: time.Now().Format("2006-01-02"),
					Amount:    p.BudgetTotal,
				},
			}
		}
		act.ActivityDates = dates
		activities = append(activities, act)
	}

	payload := IATIActivities{
		Version:           "2.03",
		GeneratedDatetime: time.Now().UTC().Format(time.RFC3339),
		Activities:        activities,
	}

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.String(http.StatusOK, xml.Header)
	enc := xml.NewEncoder(c.Writer)
	enc.Indent("", "  ")
	enc.Encode(payload)
}
