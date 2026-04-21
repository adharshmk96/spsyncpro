package workflows

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

type testActivities struct {
	logCalls      int
	simulateCalls int
}

func (a *testActivities) LogJobConfigActivity(_ context.Context, _ JobActivityInput) error {
	a.logCalls++
	return nil
}

func (a *testActivities) SimulateRunActivity(_ context.Context, _ JobActivityInput) error {
	a.simulateCalls++
	return nil
}

func TestJobExecutionWorkflow_OneTimeCompletes(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()

	acts := &testActivities{}
	env.RegisterActivityWithOptions(acts.LogJobConfigActivity, activity.RegisterOptions{Name: "LogJobConfigActivity"})
	env.RegisterActivityWithOptions(acts.SimulateRunActivity, activity.RegisterOptions{Name: "SimulateRunActivity"})

	env.ExecuteWorkflow(JobExecutionWorkflow, JobWorkflowInput{
		JobID:             "job-one-time",
		RunMode:           "ONE_TIME_AT",
		Recurrence:        "",
		StorageConfigJSON: "{}",
		FilterConfigJSON:  "{}",
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, 1, acts.logCalls)
	require.Equal(t, 1, acts.simulateCalls)
}

func TestRecurrenceInterval(t *testing.T) {
	require.Equal(t, defaultImmediateInterval, recurrenceInterval(""))
	require.Equal(t, dailySimulationInterval, recurrenceInterval("DAILY"))
	require.Equal(t, weeklySimulationInterval, recurrenceInterval("WEEKLY"))
	require.Equal(t, monthlySimulationInterval, recurrenceInterval("MONTHLY"))
}
