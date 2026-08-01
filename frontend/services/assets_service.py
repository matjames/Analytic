import os
import re
import time

ASSET_ID_PATTERN = re.compile(r'^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$')


def validate_asset_id(asset_id):
    if not isinstance(asset_id, str) or not ASSET_ID_PATTERN.fullmatch(asset_id):
        raise ValueError('Asset identifiers may contain only letters, numbers, dot, dash, and underscore.')
    if asset_id.startswith('.') or asset_id.endswith('.'):
        raise ValueError('Asset identifiers must not start or end with a separator.')
    return asset_id


def get_dashboard_asset_metadata(statgate_engine, asset_id, owner_id='tenant-alpha'):
    metadata = statgate_engine.load_dashboard_metadata(asset_id, strict=True)
    return {
        'id': asset_id,
        'asset_type': metadata.get('asset_type', 'dashboard'),
        'content_definition': metadata,
        'owner_id': owner_id,
        'version_tag': metadata.get('version_tag', '1.0.0'),
        'metadata': metadata,
    }


def build_asset_payload(dashboard_id, asset_type, content_definition, owner_id, version_tag='1.0.0'):
    validate_asset_id(dashboard_id)
    if not isinstance(asset_type, str) or not asset_type.strip():
        raise ValueError('Asset type is required.')
    if not isinstance(owner_id, str) or not owner_id.strip():
        raise ValueError('Owner identifier is required.')
    if not isinstance(version_tag, str) or not version_tag.strip():
        raise ValueError('Version tag is required.')
    if content_definition is None:
        content_definition = {}
    return {
        'id': dashboard_id,
        'asset_type': asset_type,
        'content_definition': content_definition,
        'owner_id': owner_id,
        'version_tag': version_tag,
    }


def save_dashboard_asset(statgate_engine, asset_payload):
    if not isinstance(asset_payload, dict):
        raise ValueError('Asset payload must be a dictionary.')
    if asset_payload.get('owner_id') and not isinstance(asset_payload['owner_id'], str):
        raise ValueError('Owner identifier must be a string.')

    result = statgate_engine.save_dashboard_metadata(asset_payload['id'], asset_payload['content_definition'])
    return {
        **result,
        'id': asset_payload['id'],
        'asset_type': asset_payload['asset_type'],
        'owner_id': asset_payload['owner_id'],
        'version_tag': asset_payload['version_tag'],
    }


def list_dashboard_assets(statgate_engine, dashboards_dir=None):
    dashboards_dir = dashboards_dir or os.path.join(os.path.dirname(__file__), '..', '..', 'data', 'dashboards')
    assets_list = []
    if not os.path.exists(dashboards_dir):
        return assets_list

    for filename in os.listdir(dashboards_dir):
        if not filename.endswith('.json'):
            continue

        asset_id = filename[:-5]
        metadata = statgate_engine.load_dashboard_metadata(asset_id)
        assets_list.append({
            'id': asset_id,
            'asset_type': metadata.get('asset_type', 'dashboard'),
            'content_definition': metadata,
            'owner_id': 'tenant-alpha',
            'version_tag': metadata.get('version_tag', '1.0.0'),
            'last_modified': time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime(os.path.getmtime(os.path.join(dashboards_dir, filename)))),
        })
    return assets_list
