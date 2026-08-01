import React, { useState, useEffect } from 'react'
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts'
import { DashboardApi, PublicFacilitiesApi } from '../../../helpers/api'

function AdminDashboard() {
  const [stats, setStats] = useState({
    total_facilities: 0,
    active_facilities: 0,
    pending_review: 0,
    registered_users: 0,
  })
  const [ownershipData, setOwnershipData] = useState([])
  const [levelData, setLevelData] = useState([])
  const [authorityData, setAuthorityData] = useState([])
  const [loading, setLoading] = useState(true)
  const [chartsLoading, setChartsLoading] = useState(true)

  useEffect(() => {
    const fetchStats = async () => {
      try {
        const data = await DashboardApi.getStats()
        setStats(data)
      } catch (error) {
        console.error('Failed to fetch dashboard stats:', error)
      } finally {
        setLoading(false)
      }
    }

    const fetchDistributionData = async () => {
      try {
        const [ownership, level, authority] = await Promise.all([
          PublicFacilitiesApi.getDistributionByOwnership(),
          PublicFacilitiesApi.getDistributionByLevel(),
          PublicFacilitiesApi.getDistributionByAuthority(),
        ])
        setOwnershipData(ownership)
        setLevelData(level)
        setAuthorityData(authority)
      } catch (error) {
        console.error('Failed to fetch distribution data:', error)
      } finally {
        setChartsLoading(false)
      }
    }

    fetchStats()
    fetchDistributionData()
  }, [])

  const formatNumber = (num) => {
    return new Intl.NumberFormat().format(num)
  }

  return (
    <>
      <div className="quick-stats">
        <div className="stat-card">
          <i className="bi bi-geo-alt stat-icon" />
          <div className="stat-label">Total Field Stations</div>
          <div className="stat-value">
            {loading ? '...' : formatNumber(stats.total_facilities)}
          </div>
        </div>
        <div className="stat-card">
          <i className="bi bi-check-circle stat-icon" />
          <div className="stat-label">Active Field Stations</div>
          <div className="stat-value">
            {loading ? '...' : formatNumber(stats.active_facilities)}
          </div>
        </div>
        <div className="stat-card">
          <i className="bi bi-exclamation-triangle stat-icon" />
          <div className="stat-label">Pending Review</div>
          <div className="stat-value">
            {loading ? '...' : formatNumber(stats.pending_review)}
          </div>
        </div>
        <div className="stat-card">
          <i className="bi bi-people stat-icon" />
          <div className="stat-label">Registered Users</div>
          <div className="stat-value">
            {loading ? '...' : formatNumber(stats.registered_users)}
          </div>
        </div>
      </div>

      <div style={{ display: 'flex', flexDirection: 'column', gap: '20px', width: '100%' }}>
        {/* <div className="dashboard-card" style={{ width: '100%' }}>
          <div className="card-header">
            <h3 className="card-title">Facility Distribution by Ownership</h3>
          </div>
          <div style={{ width: '100%', height: '300px', padding: '20px' }}>
            {chartsLoading ? (
              <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%' }}>
                Loading...
              </div>
            ) : (
              <ResponsiveContainer>
                <BarChart data={ownershipData}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="name" angle={-45} textAnchor="end" height={100} />
                  <YAxis />
                  <Tooltip />
                  <Legend />
                  <Bar dataKey="count" fill="#8884d8" />
                </BarChart>
              </ResponsiveContainer>
            )}
          </div>
        </div> */}

        <div className="dashboard-card" style={{ width: '100%' }}>
          <div className="card-header">
            <h3 className="card-title">Field Station Distribution by Tier</h3>
          </div>
          <div style={{ width: '100%', height: '300px', padding: '20px' }}>
            {chartsLoading ? (
              <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%' }}>
                Loading...
              </div>
            ) : (
              <ResponsiveContainer>
                <BarChart data={levelData}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="name" angle={-45} textAnchor="end" height={100} />
                  <YAxis />
                  <Tooltip />
                  <Legend />
                  <Bar dataKey="count" fill="#82ca9d" />
                </BarChart>
              </ResponsiveContainer>
            )}
          </div>
        </div>

        <div className="dashboard-card" style={{ width: '100%' }}>
          <div className="card-header">
            <h3 className="card-title">Field Station Distribution by Managing Organisation</h3>
          </div>
          <div style={{ width: '100%', height: '300px', padding: '20px' }}>
            {chartsLoading ? (
              <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%' }}>
                Loading...
              </div>
            ) : (
              <ResponsiveContainer>
                <BarChart data={authorityData}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="name" angle={-45} textAnchor="end" height={100} />
                  <YAxis />
                  <Tooltip />
                  <Legend />
                  <Bar dataKey="count" fill="#ffc658" />
                </BarChart>
              </ResponsiveContainer>
            )}
          </div>
        </div>
      </div>
    </>
  )
}

export default AdminDashboard
