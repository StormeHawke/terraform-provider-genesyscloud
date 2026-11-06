package workforcemanagement_businessunits_timeoffplan

import (
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	businessUnits "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/workforcemanagement_businessunits"
	activityCode "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/workforcemanagement_businessunits_activitycode"
	timeOffLimit "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/workforcemanagement_businessunits_timeofflimit"
	managementUnits "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/workforcemanagement_managementunits"
)

/*
   The genesyscloud_workforcemanagement_businessunits_timeoffplan_init_test.go file is used to initialize the data sources and resources
   used in testing the workforcemanagement_businessunits_timeoffplan resource.
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

	providerResources[ResourceType] = ResourceTimeOffPlan()
	providerResources[managementUnits.ResourceType] = managementUnits.ResourceManagementUnit()
	providerResources[businessUnits.ResourceType] = businessUnits.ResourceWorkforcemanagementBusinessunits()
	providerResources[activityCode.ResourceType] = activityCode.ResourceBusinessUnitActivityCode()
	providerResources[timeOffLimit.ResourceType] = timeOffLimit.ResourceTimeOffLimit()
}

// registerTestDataSources registers all data sources used in the tests.
func (r *registerTestInstance) registerTestDataSources() {
	r.datasourceMapMutex.Lock()
	defer r.datasourceMapMutex.Unlock()

	providerDataSources[ResourceType] = DataSourceTimeOffPlan()
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
