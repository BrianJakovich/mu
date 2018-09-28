package workflows

import (
	"fmt"
	"testing"

	"github.com/stelligent/mu/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewDatabaseUpserter(t *testing.T) {
	assert := assert.New(t)
	ctx := common.NewContext()
	deploy := NewDatabaseUpserter(ctx, "dev")
	assert.NotNil(deploy)
}

type mockedRdsManager struct {
	mock.Mock
}

func (m *mockedRdsManager) SetIamAuthentication(dbInstanceName string, enabled bool, dbEngine string) error {
	args := m.Called(dbInstanceName)
	return args.Error(0)
}

type mockedParamManager struct {
	mock.Mock
}

func (m *mockedParamManager) GetParam(name string) (string, error) {
	args := m.Called(name)
	return args.String(0), args.Error(1)
}
func (m *mockedParamManager) SetParam(name string, value string, kmsKey string) error {
	args := m.Called(name)
	return args.Error(0)
}
func (m *mockedParamManager) ParamVersion(name string) (int64, error) {
	args := m.Called(name)
	return args.Get(0).(int64), args.Error(1)
}

func TestDatabaseUpserter_NoName(t *testing.T) {
	assert := assert.New(t)

	stackManager := new(mockedStackManagerForService)
	rdsManager := new(mockedRdsManager)
	paramManager := new(mockedParamManager)

	config := new(common.Config)
	config.Service.Name = "foo"

	params := make(map[string]string)

	workflow := new(databaseWorkflow)
	workflow.serviceName = "foo"
	err := workflow.databaseDeployer("mu", &config.Service, params, "dev", stackManager, stackManager, rdsManager, paramManager)()
	assert.Nil(err)

	stackManager.AssertExpectations(t)
	stackManager.AssertNumberOfCalls(t, "AwaitFinalStatus", 0)
	stackManager.AssertNumberOfCalls(t, "UpsertStack", 0)

	rdsManager.AssertExpectations(t)
	rdsManager.AssertNumberOfCalls(t, "SetIamAuthentication", 0)

}

func TestDatabaseUpserter(t *testing.T) {
	assert := assert.New(t)

	stackManager := new(mockedStackManagerForService)
	stackManager.On("AwaitFinalStatus", "mu-database-foo-dev").Return(&common.Stack{Status: common.StackStatusCreateComplete, Outputs: map[string]string{"DatabaseIdentifier": "foo"}})
	stackManager.On("UpsertStack", "mu-database-foo-dev").Return(nil)

	rdsManager := new(mockedRdsManager)
	rdsManager.On("SetIamAuthentication", mock.Anything).Return(nil)

	paramManager := new(mockedParamManager)
	paramManager.On("ParamVersion", "mu-database-foo-dev-DatabaseMasterPassword").Return(int64(1), nil)

	config := new(common.Config)
	config.Service.Name = "foo"
	config.Service.Database.Name = "foo"

	params := make(map[string]string)

	workflow := new(databaseWorkflow)
	workflow.serviceName = "foo"
	err := workflow.databaseDeployer("mu", &config.Service, params, "dev", stackManager, stackManager, rdsManager, paramManager)()
	assert.Nil(err)

	stackManager.AssertExpectations(t)
	stackManager.AssertNumberOfCalls(t, "AwaitFinalStatus", 1)
	stackManager.AssertNumberOfCalls(t, "UpsertStack", 1)

	rdsManager.AssertExpectations(t)
	rdsManager.AssertNumberOfCalls(t, "SetIamAuthentication", 1)

	paramManager.AssertExpectations(t)
	paramManager.AssertNumberOfCalls(t, "ParamVersion", 1)

}

func TestDatabaseUpserter_NoPass(t *testing.T) {
	assert := assert.New(t)

	stackManager := new(mockedStackManagerForService)
	stackManager.On("AwaitFinalStatus", "mu-database-foo-dev").Return(&common.Stack{Status: common.StackStatusCreateComplete, Outputs: map[string]string{"DatabaseIdentifier": "foo"}})
	stackManager.On("UpsertStack", "mu-database-foo-dev").Return(nil)

	rdsManager := new(mockedRdsManager)
	rdsManager.On("SetIamAuthentication", mock.Anything).Return(nil)

	paramManager := new(mockedParamManager)
	paramManager.On("ParamVersion", "mu-database-foo-dev-DatabaseMasterPassword").Return(int64(0), fmt.Errorf("no password"))
	paramManager.On("SetParam", "mu-database-foo-dev-DatabaseMasterPassword", mock.Anything).Return(nil)

	config := new(common.Config)
	config.Service.Name = "foo"
	config.Service.Database.Name = "foo"

	params := make(map[string]string)

	workflow := new(databaseWorkflow)
	workflow.serviceName = "foo"
	err := workflow.databaseDeployer("mu", &config.Service, params, "dev", stackManager, stackManager, rdsManager, paramManager)()
	assert.Nil(err)

	stackManager.AssertExpectations(t)
	stackManager.AssertNumberOfCalls(t, "AwaitFinalStatus", 1)
	stackManager.AssertNumberOfCalls(t, "UpsertStack", 1)

	rdsManager.AssertExpectations(t)
	rdsManager.AssertNumberOfCalls(t, "SetIamAuthentication", 1)

	paramManager.AssertExpectations(t)
	paramManager.AssertNumberOfCalls(t, "ParamVersion", 1)
	paramManager.AssertNumberOfCalls(t, "SetParam", 1)
	assert.Equal(config.Service.Database.DatabaseConfig.MasterPasswordSSMParam, "{{resolve:ssm-secure:mu-database-foo-dev-DatabaseMasterPassword:1}}")

}

func TestDatabaseUpserter_ExistingPass(t *testing.T) {
	assert := assert.New(t)

	stackManager := new(mockedStackManagerForService)
	stackManager.On("AwaitFinalStatus", "mu-database-foo-dev").Return(&common.Stack{Status: common.StackStatusCreateComplete, Outputs: map[string]string{"DatabaseIdentifier": "foo"}})
	stackManager.On("UpsertStack", "mu-database-foo-dev").Return(nil)

	rdsManager := new(mockedRdsManager)
	rdsManager.On("SetIamAuthentication", mock.Anything).Return(nil)

	paramManager := new(mockedParamManager)
	paramManager.On("ParamVersion", "mu-database-foo-dev-DatabaseMasterPassword").Return(int64(2), nil)

	config := new(common.Config)
	config.Service.Name = "foo"
	config.Service.Database.Name = "foo"

	params := make(map[string]string)

	workflow := new(databaseWorkflow)
	workflow.serviceName = "foo"
	err := workflow.databaseDeployer("mu", &config.Service, params, "dev", stackManager, stackManager, rdsManager, paramManager)()
	assert.Nil(err)

	stackManager.AssertExpectations(t)
	stackManager.AssertNumberOfCalls(t, "AwaitFinalStatus", 1)
	stackManager.AssertNumberOfCalls(t, "UpsertStack", 1)

	rdsManager.AssertExpectations(t)
	rdsManager.AssertNumberOfCalls(t, "SetIamAuthentication", 1)

	paramManager.AssertExpectations(t)
	paramManager.AssertNumberOfCalls(t, "ParamVersion", 1)
	paramManager.AssertNumberOfCalls(t, "SetParam", 0)
	assert.Equal(config.Service.Database.DatabaseConfig.MasterPasswordSSMParam, "{{resolve:ssm-secure:mu-database-foo-dev-DatabaseMasterPassword:2}}")

}

func TestNewDatabaseUpserter_databaseRolesetUpserter(t *testing.T) {
	assert := assert.New(t)
	rolesetManager := new(mockedRolesetManagerForService)

	rolesetManager.On("UpsertCommonRoleset").Return(nil)
	rolesetManager.On("GetCommonRoleset").Return(common.Roleset{"CloudFormationRoleArn": "bar"}, nil)
	rolesetManager.On("UpsertServiceRoleset", "", "", "").Return(nil)
	rolesetManager.On("GetServiceRoleset").Return(common.Roleset{}, nil)

	workflow := new(databaseWorkflow)
	err := workflow.databaseRolesetUpserter(rolesetManager, rolesetManager, "")()
	assert.Nil(err)
	assert.Equal("bar", workflow.cloudFormationRoleArn)

	rolesetManager.AssertExpectations(t)
	rolesetManager.AssertNumberOfCalls(t, "UpsertCommonRoleset", 1)
	rolesetManager.AssertNumberOfCalls(t, "GetCommonRoleset", 1)

}

func TestDatabaseUpserter_UserDefinedSSMParam(t *testing.T) {
	assert := assert.New(t)

	stackManager := new(mockedStackManagerForService)
	stackManager.On("AwaitFinalStatus", "mu-database-foo-dev").Return(&common.Stack{Status: common.StackStatusCreateComplete, Outputs: map[string]string{"DatabaseIdentifier": "foo"}})
	stackManager.On("UpsertStack", "mu-database-foo-dev").Return(nil)

	rdsManager := new(mockedRdsManager)
	rdsManager.On("SetIamAuthentication", mock.Anything).Return(nil)

	paramManager := new(mockedParamManager)

	config := new(common.Config)
	config.Service.Name = "foo"
	config.Service.Database.Name = "foo"
	config.Service.Database.MasterPasswordSSMParam = "testDbPass:1"

	params := make(map[string]string)

	workflow := new(databaseWorkflow)
	workflow.serviceName = "foo"
	err := workflow.databaseDeployer("mu", &config.Service, params, "dev", stackManager, stackManager, rdsManager, paramManager)()
	assert.Nil(err)

	stackManager.AssertExpectations(t)
	stackManager.AssertNumberOfCalls(t, "AwaitFinalStatus", 1)
	stackManager.AssertNumberOfCalls(t, "UpsertStack", 1)

	rdsManager.AssertExpectations(t)
	rdsManager.AssertNumberOfCalls(t, "SetIamAuthentication", 1)
	assert.Equal(config.Service.Database.MasterPasswordSSMParam, "{{resolve:ssm-secure:testDbPass:1}}")
}
