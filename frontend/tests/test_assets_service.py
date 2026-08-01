from services.assets_service import (
    list_dashboard_assets,
    validate_asset_id,
    build_asset_payload,
    save_dashboard_asset,
    get_dashboard_asset_metadata,
)


def test_validate_asset_id_accepts_expected_names():
    assert validate_asset_id('dashboard_01') == 'dashboard_01'
    assert validate_asset_id('tenant.alpha') == 'tenant.alpha'


def test_validate_asset_id_rejects_invalid_values():
    try:
        validate_asset_id('bad/name')
        assert False, 'expected invalid asset id to raise ValueError'
    except ValueError:
        pass


def test_list_dashboard_assets_returns_empty_payload_when_directory_missing():
    assets = list_dashboard_assets(statgate_engine=type('Engine', (), {'load_dashboard_metadata': lambda self, asset_id: {'asset_type': 'dashboard', 'version_tag': '1.0.0'}})(), dashboards_dir='does/not/exist')
    assert assets == []


def test_build_asset_payload_and_save_dashboard_asset_return_expected_metadata():
    dashboard_id = 'tenant-alpha.dashboard_01'
    payload = build_asset_payload(dashboard_id, 'dashboard', {'widgets': []}, 'tenant-alpha', '1.0.0')
    assert payload['id'] == dashboard_id
    assert payload['asset_type'] == 'dashboard'
    assert payload['owner_id'] == 'tenant-alpha'

    engine = type('Engine', (), {'save_dashboard_metadata': lambda self, asset_id, metadata: {'status': 'saved', 'path': f'/tmp/{asset_id}.json'}})()
    result = save_dashboard_asset(engine, payload)
    assert result['id'] == dashboard_id
    assert result['asset_type'] == 'dashboard'
    assert result['owner_id'] == 'tenant-alpha'
    assert result['status'] == 'saved'


def test_get_dashboard_asset_metadata_includes_owner_and_version():
    engine = type('Engine', (), {'load_dashboard_metadata': lambda self, asset_id, strict=False: {'asset_type': 'dashboard', 'version_tag': '2.0.0'}})()
    asset = get_dashboard_asset_metadata(engine, 'tenant-alpha.dashboard_01', owner_id='tenant-alpha')
    assert asset['id'] == 'tenant-alpha.dashboard_01'
    assert asset['owner_id'] == 'tenant-alpha'
    assert asset['version_tag'] == '2.0.0'
    assert asset['asset_type'] == 'dashboard'


def test_get_dashboard_asset_metadata_raises_file_not_found_when_missing():
    def missing_loader(self, asset_id, strict=False):
        if strict:
            raise FileNotFoundError()
        return {}

    engine = type('Engine', (), {'load_dashboard_metadata': missing_loader})()
    try:
        get_dashboard_asset_metadata(engine, 'missing-dashboard', owner_id='tenant-alpha')
        assert False, 'expected FileNotFoundError when dashboard is missing in strict mode'
    except FileNotFoundError:
        pass


def test_build_asset_payload_preserves_owner_and_version_metadata():
    payload = build_asset_payload('tenant-beta.dashboard_02', 'dashboard', {'widgets': []}, 'tenant-beta', '2.0.0')

    assert payload['owner_id'] == 'tenant-beta'
    assert payload['version_tag'] == '2.0.0'
