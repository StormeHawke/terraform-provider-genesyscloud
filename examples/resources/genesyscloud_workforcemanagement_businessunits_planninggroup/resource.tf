resource "genesyscloud_workforcemanagement_businessunits_planninggroup" "voice_support" {
  business_unit_id         = genesyscloud_workforcemanagement_businessunits.example_with_settings.id
  name                     = "Voice Support"
  service_goal_template_id = genesyscloud_workforcemanagement_businessunits_servicegoaltemplate.standard_goals.id

  route_paths {
    queue_id    = genesyscloud_routing_queue.example_queue.id
    media_type  = "Voice"
    language_id = genesyscloud_routing_language.english.id
    skill_ids   = [genesyscloud_routing_skill.example_skill.id]
  }
}
