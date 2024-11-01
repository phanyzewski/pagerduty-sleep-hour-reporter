package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"time"
)

type alertReport struct {
	Alerts         []alert `json:"alerts" csv:"alerts"`
	AlertTotal     int     `json:"alert_total" csv:"alert_total"`
	OffHourTotal   int     `json:"off_hour_total" csv:"off_hour_total"`
	SleepHourTotal int     `json:"sleep_hour_total" csv:"sleep_hour_total"`
}

type responder struct {
	Name             string `json:"name" csv:"name"`
	EscalationPolicy string `json:"escalation_policy" csv:"escalation_policy"`
	OffHour          int    `json:"off_hour" csv:"off_hour"`
	SleepHour        int    `json:"sleep_hour" csv:"sleep_hour"`
}

type alert struct {
	ID         string               `json:"id" csv:"id"`
	Desc       string               `json:"desc" csv:"desc"`
	Responders map[string]responder `json:"responders" csv:"responders"`
	DateTime   time.Time            `json:"date_time" csv:"date_time"`
	URL        string               `json:"alert_url" csv:"alert_url"`
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

// func (r *alertReport) emitTableCondensed() {
// 	tw := tablewriter.NewWriter(os.Stdout)
// 	tw.SetHeader([]string{"Alert", "Description", "Team", "Responder", "Off Hour Alert", "Sleep Hour Alert"})
// 	tw.SetFooter([]string{"", "", "", fmt.Sprintf("Sleep Hours: %v", r.SleepHourTotal), fmt.Sprintf("Off Hours: %v", r.OffHourTotal), fmt.Sprintf("Alerts: %v", r.AlertTotal)})
// 	for _, a := range r.Alerts {
// 		for _, v := range a.Responders {
// 			if v.SleepHour > 0 || v.OffHour > 0 {
// 				tw.Append([]string{a.ID, a.Desc, fmt.Sprint(v.Teams), v.Name, fmt.Sprint(v.OffHour), fmt.Sprint(v.SleepHour)})
// 			}
// 		}
// 	}

// 	fmt.Printf("All Alerts from: %v to: %v\n", startMonth, endMonth)
// 	tw.Render()
// }

// func (r *alertReport) emitJson() error {
// 	formatter := json.Marshal
// 	data, err := formatter(r.Alerts)
// 	if err != nil {
// 		return err
// 	}

// 	fmt.Println(string(data))
// 	return nil
// }

func (r *alertReport) emitCsv() error {
	file, err := os.Create("report.csv")
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{"time in zone", "Alert ID", "Description", "Responder Name", "Teams", "Off Hour", "Sleep Hour", "URL"}
	if err := writer.Write(header); err != nil {
		return err
	}

	for _, a := range r.Alerts {
		for _, v := range a.Responders {
			record := []string{
				a.DateTime.String(),
				a.ID,
				a.Desc,
				v.Name,
				v.EscalationPolicy,
				fmt.Sprint(v.OffHour),
				fmt.Sprint(v.SleepHour),
				a.URL,
			}
			if err := writer.Write(record); err != nil {
				return err
			}
		}
	}

	return nil
}
