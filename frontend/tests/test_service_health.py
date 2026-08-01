from services.service_health import check_service_status, collect_service_health, list_expected_services


def test_list_expected_services_contains_known_entries():
    services = list_expected_services()

    assert any(item['id'] == 'analytics' for item in services)
    assert any(item['id'] == 'core' for item in services)
    assert any(item['id'] == 'postgres' for item in services)


def test_check_service_status_returns_unknown_for_non_http_endpoints():
    result = check_service_status({'id': 'demo', 'name': 'Demo', 'health': 'ftp://example.com'})

    assert result['status'] == 'unknown'


def test_collect_service_health_returns_list():
    result = collect_service_health()

    assert isinstance(result, list)
    assert result
