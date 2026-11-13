package workforcemanagement_managementunits_workplanrotation

import (
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/user"
	businessUnits "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/workforcemanagement_businessunits"
	managementUnits "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/workforcemanagement_managementunits"
	managementUnitAgents "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/workforcemanagement_managementunits_agents"
	workPlan "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/workforcemanagement_managementunits_workplan"
)

/*
   The genesyscloud_workforcemanagement_managementunits_workplanrotation_init_test.go file is used to initialize the data sources and resources
   used in testing the workforcemanagement_managementunits_workplanrotation resource.
*/

// providerDataSources holds a map of all registered datasources
var providerDataSources map[string]*schema.Resource

// providerResources holds a map of all registered resources
var providerResources map[string]*schema.Resource

type registerTestInstance struct {
	resourceMapMutex   sync.RWMutex
	datasourceMapMutex sync.RWMutex
}

// registerTestResources registers all resources used in the tests
func (r *registerTestInstance) registerTestResources() {
	r.resourceMapMutex.Lock()
	defer r.resourceMapMutex.Unlock()

	providerResources[ResourceType] = ResourceWorkPlanRotation()
	providerResources[workPlan.ResourceType] = workPlan.ResourceWorkPlan()
	providerResources[managementUnits.ResourceType] = managementUnits.ResourceManagementUnit()
	providerResources[managementUnitAgents.ResourceType] = managementUnitAgents.ResourceManagementUnitAgents()
	providerResources[businessUnits.ResourceType] = businessUnits.ResourceWorkforcemanagementBusinessunits()
	providerResources[user.ResourceType] = user.ResourceUser()
}

// registerTestDataSources registers all data sources used in the tests.
func (r *registerTestInstance) registerTestDataSources() {
	r.datasourceMapMutex.Lock()
	defer r.datasourceMapMutex.Unlock()

	providerDataSources[ResourceType] = DataSourceWorkPlanRotation()
}

// initTestResources initializes all test resources and data sources.
func initTestResources() {
	providerDataSources = make(map[string]*schema.Resource)
	providerResources = make(map[string]*schema.Resource)

	regInstance := &registerTestInstance{}

	regInstance.registerTestResources()
	regInstance.registerTestDataSources()
}

// TestMain is a "setup" function called by the testing framework when run the test
func TestMain(m *testing.M) {
	initTestResources()

	m.Run()
}
