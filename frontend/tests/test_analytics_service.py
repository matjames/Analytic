from services.analytics_service import build_nlq_response, infer_dataset_for_prompt, resolve_nlq_term


def test_resolve_nlq_term_matches_expected_aliases():
    term, columns = resolve_nlq_term('show me health outbreaks by region')

    assert term == 'health'
    assert 'confirmed' in columns
    assert 'deaths' in columns


def test_infer_dataset_for_prompt_uses_domain_mapping():
    assert infer_dataset_for_prompt('show me gdp by region') == 'gdp_by_region'
    assert infer_dataset_for_prompt('covid mortality trends') == 'covid_19_data'


def test_build_nlq_response_generates_contract_output():
    response = build_nlq_response('health analytics by region', 'tenant-alpha')

    assert response['semantic_label'] == 'Health'
    assert response['dataset'] == 'covid_19_data'
    assert 'WHERE tenant_id = \'tenant-alpha\'' in response['generated_sql']
    assert response['resolved_columns']
