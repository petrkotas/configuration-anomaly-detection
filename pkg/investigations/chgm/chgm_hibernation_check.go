package chgm

import (
	"sort"
	"time"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	servicelogsv1 "github.com/openshift-online/ocm-sdk-go/servicelogs/v1"
	"github.com/openshift/configuration-anomaly-detection/pkg/ocm"
)

const recentWakeupTime = 2 * time.Hour

const (
	hibernationStartEvent = "cluster_state_hibernating"
	hibernationEndEvent   = "cluster_state_ready"
)

// const hibernationOngoingEvent = "cluster_state_hibernating"
// const hibernationResumeEvent = "cluster_state_resuming"

type hibernationPeriod struct {
	HibernationDuration time.Duration
	DehibernationTime   time.Time
}

func getHibernationStatusForCluster(ocmClient ocm.Client, cluster *cmv1.Cluster) ([]*hibernationPeriod, error) {
	filter := "log_type='cluster-state-updates'"
	clusterStateUpdates, err := ocmClient.GetServiceLog(cluster, filter)
	if err != nil {
		return nil, err
	}
	return createHibernationTimeLine(clusterStateUpdates.Items().Slice()), nil
}

func createHibernationTimeLine(clusterStateUpdates []*servicelogsv1.LogEntry) []*hibernationPeriod {
	var hibernations []*hibernationPeriod

	var hibernationStartTime time.Time
	var hibernationEndTime time.Time
	sort.SliceStable(clusterStateUpdates, func(i, j int) bool {
		return clusterStateUpdates[i].Timestamp().Before(clusterStateUpdates[j].Timestamp())
	})
	for _, stateUpdate := range clusterStateUpdates {
		event := stateUpdate.Summary()
		date := stateUpdate.Timestamp()
		if event == hibernationStartEvent {
			hibernationStartTime = date
		}
		if event == hibernationEndEvent {
			if (time.Time.Equal(hibernationStartTime, time.Time{})) {
				// Cluster became ready after installation
				continue
			}
			hibernationEndTime = date
			hibernation := &hibernationPeriod{
				DehibernationTime:   hibernationEndTime,
				HibernationDuration: hibernationEndTime.Sub(hibernationStartTime),
			}
			hibernations = append(hibernations, hibernation)
		}
	}
	// Would be an ongoing hibernation
	// if (hibernationStartTime != time.Time{} && hibernationEndTime == time.Time{}) {
	// 	hibernations = append(hibernations, &HibernationPeriod{})
	// }
	return hibernations
}
