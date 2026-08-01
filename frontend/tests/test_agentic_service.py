from services.agentic_service import build_scenarios, identify_domain


def test_identify_domain_recognizes_health_goal():
    domain = identify_domain('improve covid healthcare access in district 17')

    assert domain == 'health'


def test_build_scenarios_creates_contractual_output():
    scenarios = build_scenarios('health', anomalies=[{'column': 'confirmed', 'sigma': 3.2, 'table': 'covid_19_data'}], datasets=['covid_19_data'])

    assert len(scenarios) >= 1
    assert scenarios[0]['priority'] in {'HIGH', 'CRITICAL', 'MEDIUM'}
    assert 'scenario_' in scenarios[0]['scenario_id']


def test_agentic_engine_requires_goal(monkeypatch):
    from agentic_engine import AgenticEngine

    engine = AgenticEngine()
    try:
        engine.analyze_intent('')
        assert False, 'expected ValueError for empty goal'
    except ValueError as exc:
        assert 'goal is required' in str(exc)


def test_agentic_engine_uses_dataset_discovery_resilience(monkeypatch):
    import agentic_engine
    from agentic_engine import AgenticEngine

    monkeypatch.setattr(agentic_engine, 'list_kaggle_tables', lambda: (_ for _ in ()).throw(RuntimeError('DB unavailable')))

    engine = AgenticEngine()
    report = engine.analyze_intent('Analyze cross-sector policy coordination')

    assert report['domain'] == 'general'
    assert report['dataset_discovery_error'] == 'DB unavailable'


def test_agentic_engine_identifies_tourism_domain(monkeypatch):
    import agentic_engine
    from agentic_engine import AgenticEngine

    monkeypatch.setattr(agentic_engine.statgate_engine, 'get_dataset_summary', lambda table_name: {
        'table_name': table_name,
        'row_count':  20,
        'profiling':  [],
    })

    engine = AgenticEngine()
    report = engine.analyze_intent('Revive coastal tourism and hotel occupancy')

    assert report['domain'] == 'tourism'
    assert report['dataset_candidates'] == ['hotel_bookings_raw']


def test_identify_domain_recognizes_tourism_goal():
    domain = identify_domain('boost hotel occupancy and tourism revenue')

    assert domain == 'tourism'


def test_agentic_engine_builds_scenarios_with_dataset_evidence(monkeypatch):
    import agentic_engine
    from agentic_engine import AgenticEngine

    monkeypatch.setattr(agentic_engine, 'list_kaggle_tables', lambda: ['education_by_region', 'world_development_data_imputed'])
    monkeypatch.setattr(agentic_engine.statgate_engine, 'get_dataset_summary', lambda table_name: {
        'table_name': table_name,
        'row_count': 50,
        'profiling': [
            {'column_name': 'value', 'stats': {'mean': 10.0, 'std': 1.0, 'max': 14.0}},
        ],
    })

    engine = AgenticEngine()
    report = engine.analyze_intent('Improve education outcomes in underserved regions')

    assert report['domain'] == 'education'
    assert report['dataset_discovery_error'] is None
    assert report['dataset_candidates'] == ['education_by_region', 'world_development_data_imputed']
    assert report['profile_errors'] == []
    assert len(report['scenarios']) == 3
    assert all('dataset_links' in s for s in report['scenarios'])
    assert any('education_by_region' in s['dataset_links'] for s in report['scenarios'])
    assert any('evidence' in s and 'dataset' in s['evidence'].lower() or 'Anomaly evidence' in s['evidence'] for s in report['scenarios'])
