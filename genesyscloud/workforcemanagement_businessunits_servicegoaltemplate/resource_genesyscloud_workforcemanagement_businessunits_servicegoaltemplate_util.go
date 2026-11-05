package workforcemanagement_businessunits_servicegoaltemplate

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mypurecloud/platform-client-sdk-go/v200/platformclientv2"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/util/resourcedata"
)

// SplitServiceGoalTemplateTerraformId splits the resource id, which is in the form
// <businessUnitId>/<serviceGoalTemplateId>, into the businessUnitId and serviceGoalTemplateId
func SplitServiceGoalTemplateTerraformId(id string) (businessUnitId string, templateId string) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		return id, ""
	}
	return parts[0], parts[1]
}

func firstBlock(d *schema.ResourceData, key string) (map[string]interface{}, bool) {
	list, ok := d.Get(key).([]interface{})
	if !ok || len(list) == 0 {
		return nil, false
	}
	block, ok := list[0].(map[string]interface{})
	return block, ok
}

func buildServiceLevel(d *schema.ResourceData) *platformclientv2.Buservicelevel {
	block, ok := firstBlock(d, "service_level")
	if !ok {
		return nil
	}
	serviceLevel := &platformclientv2.Buservicelevel{Include: platformclientv2.Bool(block["include"].(bool))}
	if percent, ok := block["percent"].(int); ok && serviceLevel.Include != nil && *serviceLevel.Include {
		serviceLevel.Percent = &percent
	}
	if seconds, ok := block["seconds"].(int); ok && serviceLevel.Include != nil && *serviceLevel.Include {
		serviceLevel.Seconds = &seconds
	}
	return serviceLevel
}

func buildAverageSpeedOfAnswer(d *schema.ResourceData) *platformclientv2.Buaveragespeedofanswer {
	block, ok := firstBlock(d, "average_speed_of_answer")
	if !ok {
		return nil
	}
	asa := &platformclientv2.Buaveragespeedofanswer{Include: platformclientv2.Bool(block["include"].(bool))}
	if seconds, ok := block["seconds"].(int); ok && *asa.Include {
		asa.Seconds = &seconds
	}
	return asa
}

func buildAbandonRate(d *schema.ResourceData) *platformclientv2.Buabandonrate {
	block, ok := firstBlock(d, "abandon_rate")
	if !ok {
		return nil
	}
	abandonRate := &platformclientv2.Buabandonrate{Include: platformclientv2.Bool(block["include"].(bool))}
	if percent, ok := block["percent"].(int); ok && *abandonRate.Include {
		abandonRate.Percent = &percent
	}
	return abandonRate
}

func buildServiceGoalImpact(list []interface{}) *platformclientv2.Wfmservicegoalimpact {
	if len(list) == 0 {
		return nil
	}
	block, ok := list[0].(map[string]interface{})
	if !ok {
		return nil
	}
	increaseByPercent := block["increase_by_percent"].(float64)
	decreaseByPercent := block["decrease_by_percent"].(float64)
	return &platformclientv2.Wfmservicegoalimpact{
		IncreaseByPercent: &increaseByPercent,
		DecreaseByPercent: &decreaseByPercent,
	}
}

func buildImpactOverride(d *schema.ResourceData) *platformclientv2.Servicegoaltemplateimpactoverride {
	block, ok := firstBlock(d, "impact_override")
	if !ok {
		return nil
	}
	impactOverride := &platformclientv2.Servicegoaltemplateimpactoverride{Enabled: platformclientv2.Bool(block["enabled"].(bool))}
	impactList, ok := block["impact"].([]interface{})
	if !ok || len(impactList) == 0 {
		return impactOverride
	}
	impactBlock, ok := impactList[0].(map[string]interface{})
	if !ok {
		return impactOverride
	}
	impactOverride.Impact = &platformclientv2.Wfmservicegoalimpactsettings{}
	resourcedata.BuildSDKInterfaceArrayValueIfNotNil(&impactOverride.Impact.ServiceLevel, impactBlock, "service_level", buildServiceGoalImpact)
	resourcedata.BuildSDKInterfaceArrayValueIfNotNil(&impactOverride.Impact.AverageSpeedOfAnswer, impactBlock, "average_speed_of_answer", buildServiceGoalImpact)
	resourcedata.BuildSDKInterfaceArrayValueIfNotNil(&impactOverride.Impact.AbandonRate, impactBlock, "abandon_rate", buildServiceGoalImpact)
	return impactOverride
}

func buildCreateServiceGoalTemplateRequest(d *schema.ResourceData) *platformclientv2.Createservicegoaltemplate {
	return &platformclientv2.Createservicegoaltemplate{
		Name:                 platformclientv2.String(d.Get("name").(string)),
		ServiceLevel:         buildServiceLevel(d),
		AverageSpeedOfAnswer: buildAverageSpeedOfAnswer(d),
		AbandonRate:          buildAbandonRate(d),
		ImpactOverride:       buildImpactOverride(d),
	}
}

func buildUpdateServiceGoalTemplateRequest(d *schema.ResourceData, metadata *platformclientv2.Wfmversionedentitymetadata) *platformclientv2.Updateservicegoaltemplate {
	request := &platformclientv2.Updateservicegoaltemplate{
		Name:                 platformclientv2.String(d.Get("name").(string)),
		ServiceLevel:         buildServiceLevel(d),
		AverageSpeedOfAnswer: buildAverageSpeedOfAnswer(d),
		AbandonRate:          buildAbandonRate(d),
		ImpactOverride:       buildImpactOverride(d),
	}
	if metadata != nil && metadata.Version != nil {
		request.Metadata = &platformclientv2.Wfmversionedentitymetadata{Version: metadata.Version}
	}
	return request
}

func flattenServiceLevel(serviceLevel *platformclientv2.Buservicelevel) []interface{} {
	if serviceLevel == nil {
		return nil
	}
	block := make(map[string]interface{})
	resourcedata.SetMapValueIfNotNil(block, "include", serviceLevel.Include)
	resourcedata.SetMapValueIfNotNil(block, "percent", serviceLevel.Percent)
	resourcedata.SetMapValueIfNotNil(block, "seconds", serviceLevel.Seconds)
	return []interface{}{block}
}

func flattenAverageSpeedOfAnswer(asa *platformclientv2.Buaveragespeedofanswer) []interface{} {
	if asa == nil {
		return nil
	}
	block := make(map[string]interface{})
	resourcedata.SetMapValueIfNotNil(block, "include", asa.Include)
	resourcedata.SetMapValueIfNotNil(block, "seconds", asa.Seconds)
	return []interface{}{block}
}

func flattenAbandonRate(abandonRate *platformclientv2.Buabandonrate) []interface{} {
	if abandonRate == nil {
		return nil
	}
	block := make(map[string]interface{})
	resourcedata.SetMapValueIfNotNil(block, "include", abandonRate.Include)
	resourcedata.SetMapValueIfNotNil(block, "percent", abandonRate.Percent)
	return []interface{}{block}
}

func flattenServiceGoalImpact(impact *platformclientv2.Wfmservicegoalimpact) []interface{} {
	block := make(map[string]interface{})
	resourcedata.SetMapValueIfNotNil(block, "increase_by_percent", impact.IncreaseByPercent)
	resourcedata.SetMapValueIfNotNil(block, "decrease_by_percent", impact.DecreaseByPercent)
	return []interface{}{block}
}

func flattenServiceGoalImpactSettings(settings *platformclientv2.Wfmservicegoalimpactsettings) []interface{} {
	block := make(map[string]interface{})
	resourcedata.SetMapInterfaceArrayWithFuncIfNotNil(block, "service_level", settings.ServiceLevel, flattenServiceGoalImpact)
	resourcedata.SetMapInterfaceArrayWithFuncIfNotNil(block, "average_speed_of_answer", settings.AverageSpeedOfAnswer, flattenServiceGoalImpact)
	resourcedata.SetMapInterfaceArrayWithFuncIfNotNil(block, "abandon_rate", settings.AbandonRate, flattenServiceGoalImpact)
	return []interface{}{block}
}

func flattenImpactOverride(impactOverride *platformclientv2.Servicegoaltemplateimpactoverride) []interface{} {
	if impactOverride == nil {
		return nil
	}
	block := make(map[string]interface{})
	resourcedata.SetMapValueIfNotNil(block, "enabled", impactOverride.Enabled)
	resourcedata.SetMapInterfaceArrayWithFuncIfNotNil(block, "impact", impactOverride.Impact, flattenServiceGoalImpactSettings)
	return []interface{}{block}
}

// GenerateServiceGoalTemplateResource returns terraform config for a service goal template, used in tests
func GenerateServiceGoalTemplateResource(resourceLabel string, businessUnitId string, name string, attrs ...string) string {
	return fmt.Sprintf(`resource "%s" "%s" {
	business_unit_id = %s
	name             = "%s"
	%s
}
`, ResourceType, resourceLabel, businessUnitId, name, strings.Join(attrs, "\n"))
}
