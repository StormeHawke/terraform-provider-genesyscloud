locals {
  dependencies = {
    resource = [
      "../genesyscloud_workforcemanagement_managementunits/resource.tf",
      "../genesyscloud_workforcemanagement_managementunits_workplan/resource.tf",
      "../genesyscloud_user/resource.tf",
    ]
  }
}
