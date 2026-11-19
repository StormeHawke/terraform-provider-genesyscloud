locals {
  dependencies = {
    resource = [
      "../genesyscloud_workforcemanagement_managementunits/resource.tf",
      "../genesyscloud_workforcemanagement_businessunits_activitycode/resource.tf",
      "../genesyscloud_workforcemanagement_businessunits_timeofflimit/resource.tf",
    ]
  }
}
