package main

import (
	"bytes"
	"encoding/json"
	"log"
	"os/exec"

	"github.com/prometheus/client_golang/prometheus"
)

type CEPHCollector struct {
	cephPath string
	descs    map[string]*prometheus.Desc
}

func newDesc(ns, name, help string, labels []string) *prometheus.Desc {
	return prometheus.NewDesc(ns+"_"+name, help, labels, nil)
}

func NewCEPHCollector(cephPath string) *CEPHCollector {
	ns := "ceph"
	descs := map[string]*prometheus.Desc{
		"total_bytes":                newDesc(ns, "total_bytes", "cluster total bytes", []string{}),
		"total_avail_bytes":          newDesc(ns, "total_avail_bytes", "cluster total avail bytes", []string{}),
		"total_used_raw_bytes":       newDesc(ns, "total_used_raw_bytes", "cluster total used bytes", []string{}),
		"total_used_raw_ratio":       newDesc(ns, "total_used_raw_ratio", "cluster total used ratio", []string{}),
		"num_osds":                   newDesc(ns, "num_osds", "number osds", []string{}),
		"pool_per_osd":               newDesc(ns, "pool_per_osd", "number osds per pool", []string{}),
		"by_class_total_bytes":       newDesc(ns, "stats_by_class_total_bytes", "stats by class total bytes", []string{"class"}),
		"by_class_total_avail_bytes": newDesc(ns, "stats_by_class_total_avail_bytes", "stats by class total avail bytes", []string{"class"}),
		"by_class_used_raw_bytes":    newDesc(ns, "stats_by_class_used_raw_bytes", "stats by class used bytes", []string{"class"}),
		"by_class_used_raw_ratio":    newDesc(ns, "stats_by_class_used_raw_ratio", "stats by class used ratio", []string{"class"}),
		"osd_total_bytes":            newDesc(ns, "osd_total_bytes", "osd total bytes", []string{"name", "class"}),
		"osd_used_bytes":             newDesc(ns, "osd_used_bytes", "osd used bytes", []string{"name", "class"}),
		"osd_data_used_bytes":        newDesc(ns, "osd_data_used_bytes", "osd data used bytes", []string{"name", "class"}),
		"osd_omap_used_bytes":        newDesc(ns, "osd_omap_used_bytes", "osd omap used bytes", []string{"name", "class"}),
		"osd_meta_used_bytes":        newDesc(ns, "osd_meta_used_bytes", "osd meta used bytes", []string{"name", "class"}),
		"osd_pgs":                    newDesc(ns, "osd_pgs", "number pgs in osd", []string{"name", "class"}),
		"osd_status":                 newDesc(ns, "osd_status", "osd status", []string{"name", "class"}),
	}
	return &CEPHCollector{
		cephPath: cephPath,
		descs:    descs,
	}
}

func (c *CEPHCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, m := range c.descs {
		ch <- m
	}
}

func (c *CEPHCollector) ceph(args ...string) ([]byte, error) {
	params := []string{
		"-f",
		"json",
	}
	for _, a := range args {
		params = append(params, a)
	}
	cmd := exec.Command(c.cephPath, params...)
	data, err := cmd.Output()
	if err != nil {
		return data, err
	}
	return data, nil
}

func (c *CEPHCollector) collect(ch chan<- prometheus.Metric, key string, value float64, labels ...string) {
	ch <- prometheus.MustNewConstMetric(
		c.descs[key],
		prometheus.GaugeValue,
		value,
		labels...,
	)
}

func (c *CEPHCollector) Collect(ch chan<- prometheus.Metric) {
	c.collectClusterUsage(ch)
	c.collectOsdsUsage(ch)
}

type SummaryStats struct {
	TotalBytes         int64   `json:"total_bytes"`
	AvailBytes         int64   `json:"total_avail_bytes"`
	UsedBytes          int64   `json:"total_used_bytes"`
	UsedRawBytes       int64   `json:"total_used_raw_bytes"`
	UsedRawRatio       float64 `json:"total_used_raw_ratio"`
	NumOsds            int     `json:"num_osds"`
	NumPerPoolOsds     int     `json:"num_per_pool_osds"`
	NumPerPoolOmapOsds int     `json:"num_per_pool_omap_osds"`
}

type ClassStats struct {
	TotalBytes   int64   `json:"total_bytes"`
	AvailBytes   int64   `json:"total_avail_bytes"`
	UsedBytes    int64   `json:"total_used_bytes"`
	UsedRawBytes int64   `json:"total_used_raw_bytes"`
	UsedRawRatio float64 `json:"total_used_raw_ratio"`
}

type PoolStatsDetail struct {
	Stored        int64   `json:"stored"`
	Objects       int64   `json:"objects"`
	UsedKB        int64   `json:"kb_used"`
	UsedBytes     int64   `json:"bytes_used"`
	UsedRatio     float64 `json:"percent_used"`
	MaxAvailBytes int64   `json:"max_avail"`
}

type PoolStats struct {
	Id    int             `json:"id"`
	Name  string          `json:"name"`
	Stats PoolStatsDetail `json:"stats"`
}

type ClusterUsage struct {
	Stats        SummaryStats          `json:"stats"`
	StatsByClass map[string]ClassStats `json:"stats_by_class"`
	Pools        []PoolStats           `json:"pools"`
}

func (c *CEPHCollector) collectClusterUsage(ch chan<- prometheus.Metric) {
	output, err := c.ceph("df")
	if err != nil {
		log.Printf("[Error]: collect cluster usage got error: %v", err)
	}
	output = bytes.TrimSpace(output)
	var cu ClusterUsage
	err = json.Unmarshal(output, &cu)
	if err != nil {
		log.Printf("[Error]: parse ceph status report output got error: %v", err)
		return
	}

	c.collect(ch, "total_bytes", float64(cu.Stats.TotalBytes))
	c.collect(ch, "total_avail_bytes", float64(cu.Stats.AvailBytes))
	c.collect(ch, "total_used_raw_bytes", float64(cu.Stats.UsedRawBytes))
	c.collect(ch, "total_used_raw_ratio", float64(cu.Stats.UsedRawRatio))
	c.collect(ch, "num_osds", float64(cu.Stats.NumOsds))
	c.collect(ch, "pool_per_osd", float64(cu.Stats.NumPerPoolOsds))

	for class, cs := range cu.StatsByClass {
		c.collect(ch, "by_class_total_bytes", float64(cs.TotalBytes), class)
		c.collect(ch, "by_class_total_avail_bytes", float64(cs.AvailBytes), class)
		c.collect(ch, "by_class_used_raw_bytes", float64(cs.UsedRawBytes), class)
		c.collect(ch, "by_class_used_raw_ratio", float64(cs.UsedRawRatio), class)
	}
}

type OsdStats struct {
	Id          int     `json:"id"`
	Class       string  `json:"device_class"`
	Name        string  `json:"name"`
	TypeName    string  `json:"type"`
	CrushWeight float64 `json:"crush_weight"`
	Reweight    float64 `json:"reweight"`
	TotalKB     int64   `json:"kb"`
	UsedKB      int64   `json:"kb_used"`
	DataUsedKB  int64   `json:"kb_used_data"`
	OmapUsedKB  int64   `json:"kb_used_omap"`
	MetaUsedKB  int64   `json:"kb_used_meta"`
	AvailKB     int64   `json:"kb_avail"`
	UsedRatio   float64 `json:"utilization"`
	Pgs         int     `json:"pgs"`
	Status      string  `json:"status"`
}

type OsdsStats struct {
	Nodes []OsdStats `json:"nodes"`
}

func (c *CEPHCollector) collectOsdsUsage(ch chan<- prometheus.Metric) {
	output, err := c.ceph("osd", "df")
	if err != nil {
		log.Printf("[Error]: collect osds usage got error: %v", err)
	}
	output = bytes.TrimSpace(output)
	var ou OsdsStats
	err = json.Unmarshal(output, &ou)
	if err != nil {
		log.Printf("[Error]: parse osds status report output got error: %v", err)
		return
	}

	for _, os := range ou.Nodes {
		name := os.Name
		class := os.Class
		c.collect(ch, "osd_total_bytes", float64(os.TotalKB*1024), name, class)
		c.collect(ch, "osd_used_bytes", float64(os.UsedKB*1024), name, class)
		c.collect(ch, "osd_data_used_bytes", float64(os.DataUsedKB*1024), name, class)
		c.collect(ch, "osd_omap_used_bytes", float64(os.OmapUsedKB*1024), name, class)
		c.collect(ch, "osd_meta_used_bytes", float64(os.MetaUsedKB*1024), name, class)
		c.collect(ch, "osd_pgs", float64(os.Pgs), name, class)
		status := 0
		if os.Status == "up" {
			status = 1
		}
		c.collect(ch, "osd_status", float64(status), name, class)
	}

}
