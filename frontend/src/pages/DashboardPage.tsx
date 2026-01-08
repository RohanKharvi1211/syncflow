import { useEffect, useState } from 'react';
import { useQuery } from 'react-query';
import { useAuthStore } from '@shared/store/authStore';
import { useCompanyStore } from '@shared/store/companyStore';
import { pipelineApi } from '@domains/pipelines/adapters/pipelineApi';
import { Pipeline } from '@domains/pipelines/entities/Pipeline';

export function DashboardPage() {
  const { user } = useAuthStore();
  const { selectedCompany } = useCompanyStore();
  const [stats, setStats] = useState({
    totalPipelines: 0,
    activePipelines: 0,
    pausedPipelines: 0,
    lastSync: null as string | null,
  });

  const { data: pipelines } = useQuery<Pipeline[]>(
    ['pipelines', selectedCompany?.id],
    () => pipelineApi.getPipelines(selectedCompany?.id || user?.company_id || ''),
    {
      enabled: !!selectedCompany || !!user?.company_id,
      onSuccess: (data) => {
        setStats({
          totalPipelines: data.length,
          activePipelines: data.filter((p) => p.status === 'ACTIVE').length,
          pausedPipelines: data.filter((p) => p.status === 'PAUSED').length,
          lastSync: data[0]?.checkpoint?.last_sync_time || null,
        });
      },
    }
  );

  const statCards = [
    {
      label: 'Total Pipelines',
      value: stats.totalPipelines,
      icon: '🔄',
      color: 'blue',
    },
    {
      label: 'Active',
      value: stats.activePipelines,
      icon: '✅',
      color: 'green',
    },
    {
      label: 'Paused',
      value: stats.pausedPipelines,
      icon: '⏸️',
      color: 'yellow',
    },
    {
      label: 'Last Sync',
      value: stats.lastSync
        ? new Date(stats.lastSync).toLocaleString()
        : 'Never',
      icon: '🕐',
      color: 'gray',
    },
  ];

  return (
    <div>
      <h1 className="text-3xl font-bold text-gray-900 mb-8">Dashboard</h1>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
        {statCards.map((stat) => (
          <div
            key={stat.label}
            className="bg-white rounded-lg shadow p-6 border-l-4 border-primary-500"
          >
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-gray-600">{stat.label}</p>
                <p className="text-2xl font-bold text-gray-900 mt-2">{stat.value}</p>
              </div>
              <span className="text-4xl">{stat.icon}</span>
            </div>
          </div>
        ))}
      </div>

      <div className="bg-white rounded-lg shadow">
        <div className="p-6 border-b">
          <h2 className="text-xl font-semibold text-gray-900">Recent Pipelines</h2>
        </div>
        <div className="p-6">
          {pipelines && pipelines.length > 0 ? (
            <div className="space-y-4">
              {pipelines.slice(0, 5).map((pipeline) => (
                <div
                  key={pipeline.id}
                  className="flex items-center justify-between p-4 border rounded-lg hover:bg-gray-50"
                >
                  <div>
                    <h3 className="font-medium text-gray-900">
                      {pipeline.source_object?.connection?.app?.display_name} →{' '}
                      {pipeline.destination_object?.connection?.app?.display_name}
                    </h3>
                    <p className="text-sm text-gray-500">
                      Status: {pipeline.status} • Every {pipeline.schedule_interval} min
                    </p>
                  </div>
                  <span
                    className={`px-3 py-1 rounded-full text-xs font-medium ${
                      pipeline.status === 'ACTIVE'
                        ? 'bg-green-100 text-green-800'
                        : 'bg-yellow-100 text-yellow-800'
                    }`}
                  >
                    {pipeline.status}
                  </span>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-gray-500 text-center py-8">No pipelines yet</p>
          )}
        </div>
      </div>
    </div>
  );
}


