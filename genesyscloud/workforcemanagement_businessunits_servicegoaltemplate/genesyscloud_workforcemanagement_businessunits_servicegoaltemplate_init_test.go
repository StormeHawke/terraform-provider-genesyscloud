package workforcemanagement_businessunits_servicegoaltemplate

import (
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	businessUnits "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/workforcemanagement_businessunits"
)

/*
   The genesyscloud_workforcemanagement_businessunits_servicegoaltemplate_init_test.go file is used to initialize the data sources and resources
   used in testing the workforcemanagement_businessunits_servicegoaltemplate resource.
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

	providerResources[ResourceType] = ResourceServiceGoalTemplate()
	providerResources[businessUnits.ResourceType] = businessUnits.ResourceWorkforcemanagementBusinessunits()
}

// registerTestDataSources registers all data sources used in the tests.
func (r *registerTestInstance) registerTestDataSources() {
	r.datasourceMapMutex.Lock()
	defer r.datasourceMapMutex.Unlock()

	providerDataSources[ResourceType] = DataSourceServiceGoalTemplate()
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
