import os
import json
import tempfile
import importlib


def reload_module():
    import schema_healer
    return importlib.reload(schema_healer)


def test_detect_drift_returns_mapping_for_renamed_columns(monkeypatch, tmp_path):
    schema_healer = reload_module()

    old_schema = [
        {"column_name": "country", "data_type": "text", "is_nullable": "YES"},
        {"column_name": "confirmed", "data_type": "integer", "is_nullable": "YES"},
    ]
    new_schema = [
        {"column_name": "country_per_region", "data_type": "text", "is_nullable": "YES"},
        {"column_name": "confirmed", "data_type": "integer", "is_nullable": "YES"},
    ]

    monkeypatch.setattr(schema_healer, "_load_snapshot", lambda table_name: old_schema)
    monkeypatch.setattr(schema_healer, "_save_snapshot", lambda table_name, schema: None)
    monkeypatch.setattr(schema_healer, "get_table_schema", lambda table_name: new_schema)

    report = schema_healer.detect_drift("covid_19_data")

    assert report["drifted"]
    assert report["added_cols"] == ["country_per_region"]
    assert report["removed_cols"] == ["country"]
    assert report["auto_mapping"] == {"country": "country_per_region"}


def test_detect_drift_infers_synonym_renames(monkeypatch, tmp_path):
    schema_healer = reload_module()

    old_schema = [
        {"column_name": "case_count", "data_type": "integer", "is_nullable": "YES"},
        {"column_name": "region", "data_type": "text", "is_nullable": "YES"},
    ]
    new_schema = [
        {"column_name": "confirmed", "data_type": "integer", "is_nullable": "YES"},
        {"column_name": "country", "data_type": "text", "is_nullable": "YES"},
    ]

    monkeypatch.setattr(schema_healer, "_load_snapshot", lambda table_name: old_schema)
    monkeypatch.setattr(schema_healer, "_save_snapshot", lambda table_name, schema: None)
    monkeypatch.setattr(schema_healer, "get_table_schema", lambda table_name: new_schema)

    report = schema_healer.detect_drift("covid_19_data")

    assert report["drifted"]
    assert report["auto_mapping"] == {"case_count": "confirmed"}
    assert report["renamed_cols"] == [{"from": "case_count", "to": "confirmed"}]


def test_heal_assets_updates_dashboard_json_with_column_renames(tmp_path, monkeypatch):
    schema_healer = reload_module()
    dashboards_dir = tmp_path / "dashboards"
    dashboards_dir.mkdir()

    dashboard_file = dashboards_dir / "tenant-alpha.example.json"
    original = {"filters": {"country": "US"}, "metrics": ["confirmed"]}
    dashboard_file.write_text(json.dumps(original))

    monkeypatch.setattr(schema_healer, "DASHBOARDS_DIR", str(dashboards_dir))

    report = {"drifted": True, "auto_mapping": {"country": "country_per_region"}}
    result = schema_healer.heal_assets(report)

    assert result["patched"] == 1
    assert result["assets"] == [dashboard_file.name]

    updated_content = json.loads(dashboard_file.read_text())
    assert updated_content["filters"]["country_per_region"] == "US"
    assert "country" not in updated_content["filters"]


def test_snapshot_all_tables_populates_snapshots(tmp_path, monkeypatch):
    schema_healer = reload_module()
    monkeypatch.setattr(schema_healer, "SNAPSHOT_DIR", str(tmp_path / "snapshots"))

    import kaggle_connector
    monkeypatch.setattr(kaggle_connector, "list_kaggle_tables", lambda: ["covid_19_data", "other_table"])
    monkeypatch.setattr(schema_healer, "get_table_schema", lambda table_name: [
        {"column_name": "id", "data_type": "integer", "is_nullable": "NO"},
        {"column_name": "value", "data_type": "double precision", "is_nullable": "YES"}
    ])

    results = schema_healer.snapshot_all_tables()

    assert results == {"covid_19_data": "ok", "other_table": "ok"}
    assert (tmp_path / "snapshots" / "covid_19_data.json").exists()
    assert (tmp_path / "snapshots" / "other_table.json").exists()


def test_snapshot_all_tables_reports_missing_table_errors(monkeypatch, tmp_path):
    schema_healer = reload_module()
    monkeypatch.setattr(schema_healer, "SNAPSHOT_DIR", str(tmp_path / "snapshots"))
    import kaggle_connector
    monkeypatch.setattr(kaggle_connector, "list_kaggle_tables", lambda: ["missing_table"])
    monkeypatch.setattr(schema_healer, "get_table_schema", lambda table_name: (_ for _ in ()).throw(kaggle_connector.TableNotFoundError(f'Table "{table_name}" not found in schema "ml_staging".')))

    results = schema_healer.snapshot_all_tables()

    assert "missing_table" in results
    assert "not found in schema" in results["missing_table"]
