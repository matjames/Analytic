import services.dataset_metadata as dataset_metadata


def test_upsert_dataset_metadata_uses_scoped_insert_and_update(monkeypatch):
    executed = []

    class FakeCursor:
        def __init__(self, cursor_factory=None):
            self.cursor_factory = cursor_factory
            self.fetch_count = 0

        def __enter__(self):
            return self

        def __exit__(self, *_args):
            return False

        def execute(self, query, args=None):
            executed.append((query, args))

        def fetchone(self):
            self.fetch_count += 1
            if self.fetch_count == 1:
                return None
            return {
                'table_name': 'covid_19_data',
                'tenant_id': 'tenant-beta',
                'workspace_id': 'workspace_1',
            }

    class FakeConn:
        def cursor(self, cursor_factory=None):
            return FakeCursor(cursor_factory)

        def commit(self):
            pass

        def close(self):
            pass

    monkeypatch.setattr(dataset_metadata, '_get_conn', lambda: FakeConn())

    result = dataset_metadata.upsert_dataset_metadata(
        'covid_19_data',
        owner_id='owner-1',
        row_count=12,
        col_count=2,
        tenant_id='tenant-beta',
        workspace_id='workspace_1',
    )

    assert result['tenant_id'] == 'tenant-beta'
    assert result['workspace_id'] == 'workspace_1'
    assert any('idx_dataset_metadata_scope_unique' in query for query, _args in executed)
    assert any('ON CONFLICT DO NOTHING' in query for query, _args in executed)
    assert any('workspace_id IS NOT DISTINCT FROM %s' in query for query, _args in executed)
