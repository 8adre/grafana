// +build integration

package ngalert

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/grafana/grafana/pkg/services/ngalert/eval"
	"github.com/grafana/grafana/pkg/services/sqlstore"
	"github.com/grafana/grafana/pkg/setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreatingAlertDefinition(t *testing.T) {
	t.Run("should fail gracefully when creating alert definition with invalid relative time range", func(t *testing.T) {
		ng := setupTestEnv(t)
		q := saveAlertDefinitionCommand{
			OrgID: 1,
			Name:  "something completely different",
			Condition: condition{
				RefID: "B",
				QueriesAndExpressions: []eval.AlertQuery{
					{
						Model: json.RawMessage(`{
							"datasource": "__expr__",
							"type":"math",
							"expression":"2 + 3 > 1"
						}`),
						RefID: "B",
					},
				},
			},
		}
		err := ng.saveAlertDefinition(&q)
		require.NoError(t, err)
	})

}

func TestUpdatingAlertDefinition(t *testing.T) {
	t.Run("zero rows affected when updating unknown alert", func(t *testing.T) {
		ng := setupTestEnv(t)

		q := updateAlertDefinitionCommand{
			ID:    1,
			OrgID: 1,
			Name:  "something completely different",
			Condition: condition{
				RefID: "A",
				QueriesAndExpressions: []eval.AlertQuery{
					{
						Model: json.RawMessage(`{
							"datasource": "__expr__",
							"type":"math",
							"expression":"2 + 2 > 1"
						}`),
						RefID: "A",
						RelativeTimeRange: eval.RelativeTimeRange{
							From: eval.Duration(time.Duration(5) * time.Hour),
							To:   eval.Duration(time.Duration(3) * time.Hour),
						},
					},
				},
			},
		}

		err := ng.updateAlertDefinition(&q)
		require.NoError(t, err)
		assert.Equal(t, int64(0), q.RowsAffected)
	})

	t.Run("updating successfully existing alert", func(t *testing.T) {
		ng := setupTestEnv(t)
		alertDefinition := createTestAlertDefinition(t, ng)

		q := updateAlertDefinitionCommand{
			ID:    (*alertDefinition).Id,
			OrgID: 1,
			Name:  "something completely different",
			Condition: condition{
				RefID: "B",
				QueriesAndExpressions: []eval.AlertQuery{
					{
						Model: json.RawMessage(`{
							"datasource": "__expr__",
							"type":"math",
							"expression":"2 + 3 > 1"
						}`),
						RefID: "B",
						RelativeTimeRange: eval.RelativeTimeRange{
							From: eval.Duration(5 * time.Hour),
							To:   eval.Duration(3 * time.Hour),
						},
					},
				},
			},
		}

		err := ng.updateAlertDefinition(&q)
		require.NoError(t, err)
		assert.Equal(t, int64(1), q.RowsAffected)
		assert.Equal(t, int64(1), q.Result.Id)

		getAlertDefinitionByIDQuery := getAlertDefinitionByIDQuery{ID: (*alertDefinition).Id}
		err = ng.getAlertDefinitionByID(&getAlertDefinitionByIDQuery)
		require.NoError(t, err)
		assert.Equal(t, "something completely different", getAlertDefinitionByIDQuery.Result.Name)
		assert.Equal(t, "B", getAlertDefinitionByIDQuery.Result.Condition)
		assert.Equal(t, 1, len(getAlertDefinitionByIDQuery.Result.Data))
	})
}

func TestDeletingAlertDefinition(t *testing.T) {
	t.Run("zero rows affected when deleting unknown alert", func(t *testing.T) {
		ng := setupTestEnv(t)

		q := deleteAlertDefinitionByIDQuery{
			ID:    1,
			OrgID: 1,
		}

		err := ng.deleteAlertDefinitionByID(&q)
		require.NoError(t, err)
		assert.Equal(t, int64(0), q.RowsAffected)
	})

	t.Run("deleting successfully existing alert", func(t *testing.T) {
		ng := setupTestEnv(t)
		alertDefinition := createTestAlertDefinition(t, ng)

		q := deleteAlertDefinitionByIDQuery{
			ID:    (*alertDefinition).Id,
			OrgID: 1,
		}

		err := ng.deleteAlertDefinitionByID(&q)
		require.NoError(t, err)
		assert.Equal(t, int64(1), q.RowsAffected)
	})
}

func setupTestEnv(t *testing.T) AlertNG {
	sqlStore := sqlstore.InitTestDB(t)
	cfg := setting.Cfg{}
	cfg.FeatureToggles = map[string]bool{"ngalert": true}
	ng := AlertNG{
		SQLStore: sqlStore,
		Cfg:      &cfg,
	}
	return ng
}

func createTestAlertDefinition(t *testing.T, ng AlertNG) *AlertDefinition {
	cmd := saveAlertDefinitionCommand{
		OrgID: 1,
		Name:  "an alert definition",
		Condition: condition{
			RefID: "A",
			QueriesAndExpressions: []eval.AlertQuery{
				{
					Model: json.RawMessage(`{
						"datasource": "__expr__",
						"type":"math",
						"expression":"2 + 2 > 1"
					}`),
					RelativeTimeRange: eval.RelativeTimeRange{
						From: eval.Duration(5 * time.Hour),
						To:   eval.Duration(3 * time.Hour),
					},
					RefID: "A",
				},
			},
		},
	}
	err := ng.saveAlertDefinition(&cmd)
	require.NoError(t, err)
	return cmd.Result
}

func TestAlertInstanceOperations(t *testing.T) {
	ng := setupTestEnv(t)

	t.Run("can save and read new alert instance", func(t *testing.T) {
		saveCmd := &saveAlertInstanceCommand{
			OrgID:             1,
			AlertDefinitionID: 1,
			State:             InstanceStateFiring,
			Labels:            InstanceLabels{"test": "testValue"},
		}
		err := ng.saveAlertInstance(saveCmd)
		require.NoError(t, err)

		getCmd := &getAlertInstanceCommand{
			OrgID:             1,
			AlertDefinitionID: 1,
			Labels:            InstanceLabels{"test": "testValue"},
		}

		err = ng.getAlertInstance(getCmd)
		require.NoError(t, err)

		spew.Dump(getCmd.Result)

		require.Equal(t, saveCmd.Labels, getCmd.Result.Labels)

	})

	t.Run("can save and read new alert instance with no labels", func(t *testing.T) {
		saveCmd := &saveAlertInstanceCommand{
			OrgID:             1,
			AlertDefinitionID: 2,
			State:             InstanceStateNormal,
		}
		err := ng.saveAlertInstance(saveCmd)
		require.NoError(t, err)

		getCmd := &getAlertInstanceCommand{
			OrgID:             1,
			AlertDefinitionID: 2,
		}

		err = ng.getAlertInstance(getCmd)
		require.NoError(t, err)

		spew.Dump(getCmd.Result)

		require.Equal(t, saveCmd.AlertDefinitionID, getCmd.Result.AlertDefinitionID)
		require.Equal(t, saveCmd.Labels, getCmd.Result.Labels)

	})

	t.Run("can save two instances with same id and different labels", func(t *testing.T) {
		saveCmdOne := &saveAlertInstanceCommand{
			OrgID:             1,
			AlertDefinitionID: 3,
			State:             InstanceStateFiring,
			Labels:            InstanceLabels{"test": "testValue"},
		}

		err := ng.saveAlertInstance(saveCmdOne)
		require.NoError(t, err)

		saveCmdTwo := &saveAlertInstanceCommand{
			OrgID:             1,
			AlertDefinitionID: 3,
			State:             InstanceStateFiring,
			Labels:            InstanceLabels{"test": "meow"},
		}
		err = ng.saveAlertInstance(saveCmdTwo)
		require.NoError(t, err)
	})

	t.Run("can list all added instances in org", func(t *testing.T) {
		listCommand := &listAlertInstancesCommand{
			OrgID: 1,
		}

		err := ng.listAlertInstances(listCommand)
		require.NoError(t, err)

		require.Len(t, listCommand.Result, 4)
	})

	t.Run("can list all added instances in org filtered by current state", func(t *testing.T) {
		listCommand := &listAlertInstancesCommand{
			OrgID: 1,
			State: InstanceStateNormal,
		}

		err := ng.listAlertInstances(listCommand)
		require.NoError(t, err)

		require.Len(t, listCommand.Result, 1)
	})

}
