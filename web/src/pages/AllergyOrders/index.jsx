/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
  Button,
  Card,
  Input,
  Pagination,
  Select,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import CardTable from '../../components/common/ui/CardTable';
import { API, showError } from '../../helpers';

const paymentStatusOptions = [
  { label: '全部支付状态', value: '' },
  { label: '待支付', value: 'pending' },
  { label: '已支付', value: 'paid' },
  { label: '已退款', value: 'refunded' },
  { label: '已取消', value: 'cancelled' },
];

const orderStatusOptions = [
  { label: '全部履约状态', value: '' },
  { label: '待支付', value: 'pending_payment' },
  { label: '已支付', value: 'paid' },
  { label: '备货中', value: 'kit_preparing' },
  { label: '采样盒已寄出', value: 'kit_shipped' },
  { label: '样本回寄中', value: 'sample_returning' },
  { label: '样本已签收', value: 'sample_received' },
  { label: '检测中', value: 'in_testing' },
  { label: '报告已就绪', value: 'report_ready' },
  { label: '已完成', value: 'completed' },
  { label: '已取消', value: 'cancelled' },
];

function getStatusMeta(status) {
  const meta = {
    pending: { color: 'grey', label: '待支付' },
    paid: { color: 'green', label: '已支付' },
    refunded: { color: 'orange', label: '已退款' },
    cancelled: { color: 'red', label: '已取消' },
    pending_payment: { color: 'grey', label: '待支付' },
    kit_preparing: { color: 'blue', label: '备货中' },
    kit_shipped: { color: 'cyan', label: '采样盒已寄出' },
    sample_returning: { color: 'cyan', label: '样本回寄中' },
    sample_received: { color: 'blue', label: '样本已签收' },
    in_testing: { color: 'blue', label: '检测中' },
    report_ready: { color: 'green', label: '报告已就绪' },
    completed: { color: 'green', label: '已完成' },
  };
  return meta[status] || { color: 'grey', label: status || '-' };
}

function formatDateTime(value) {
  if (!value) return '-';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString();
}

function getRequestErrorMessage(error, fallback) {
  return error?.response?.data?.message || error?.message || fallback;
}

const AllergyOrders = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [items, setItems] = useState([]);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [filters, setFilters] = useState({
    orderNo: '',
    email: '',
    paymentStatus: '',
    orderStatus: '',
  });

  const loadOrders = async (
    nextPage = page,
    nextPageSize = pageSize,
    nextFilters = filters,
  ) => {
    setLoading(true);
    try {
      const res = await API.get('/api/admin/orders', {
        params: {
          p: nextPage,
          page_size: nextPageSize,
          order_no: nextFilters.orderNo,
          email: nextFilters.email,
          payment_status: nextFilters.paymentStatus,
          order_status: nextFilters.orderStatus,
        },
      });
      if (!res.data.success) {
        showError(res.data.message);
        return;
      }
      const payload = res.data.data || {};
      setItems(payload.items || []);
      setTotal(payload.total || 0);
      setPage(payload.page || nextPage);
      setPageSize(payload.page_size || nextPageSize);
    } catch (error) {
      showError(getRequestErrorMessage(error, '获取过敏订单失败'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadOrders(1, pageSize, filters);
  }, []);

  const columns = [
    {
      title: t('订单号'),
      dataIndex: 'order_no',
      key: 'order_no',
      render: (value, record) => (
        <div className='flex flex-col'>
          <span className='font-semibold'>{value}</span>
          <span className='text-xs text-semi-color-text-2'>
            {record.service_name || '-'}
          </span>
        </div>
      ),
    },
    {
      title: t('收件人'),
      dataIndex: 'recipient_name',
      key: 'recipient_name',
      render: (value, record) => (
        <div className='flex flex-col'>
          <span>{value || '-'}</span>
          <span className='text-xs text-semi-color-text-2'>
            {record.recipient_email || '-'}
          </span>
        </div>
      ),
    },
    {
      title: t('支付状态'),
      dataIndex: 'payment_status',
      key: 'payment_status',
      render: (value) => {
        const meta = getStatusMeta(value);
        return (
          <Tag color={meta.color} shape='circle' type='light'>
            {meta.label}
          </Tag>
        );
      },
    },
    {
      title: t('履约状态'),
      dataIndex: 'order_status',
      key: 'order_status',
      render: (value) => {
        const meta = getStatusMeta(value);
        return (
          <Tag color={meta.color} shape='circle' type='light'>
            {meta.label}
          </Tag>
        );
      },
    },
    {
      title: t('支付时间'),
      dataIndex: 'paid_at',
      key: 'paid_at',
      render: (value) => formatDateTime(value),
    },
    {
      title: t('创建时间'),
      dataIndex: 'created_at',
      key: 'created_at',
      render: (value) => formatDateTime(value),
    },
    {
      title: t('操作'),
      key: 'action',
      render: (_, record) => (
        <Button
          type='primary'
          theme='outline'
          size='small'
          onClick={() => navigate(`/console/allergy-orders/${record.order_id}`)}
        >
          {t('查看详情')}
        </Button>
      ),
    },
  ];

  return (
    <div className='mt-[60px] px-2'>
      <div className='mb-4 flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between'>
        <div>
          <Typography.Title heading={4} className='!mb-1'>
            {t('过敏订单')}
          </Typography.Title>
          <Typography.Text type='secondary'>
            {t('查看支付状态、履约进度、报告上传与发送情况。')}
          </Typography.Text>
        </div>
        <Button theme='outline' onClick={() => loadOrders(page, pageSize, filters)}>
          {t('刷新')}
        </Button>
      </div>

      <Card className='!rounded-2xl shadow-sm border-0'>
        <div className='grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-4'>
          <Input
            placeholder={t('按订单号搜索')}
            value={filters.orderNo}
            onChange={(value) =>
              setFilters((prev) => ({ ...prev, orderNo: value }))
            }
          />
          <Input
            placeholder={t('按邮箱搜索')}
            value={filters.email}
            onChange={(value) =>
              setFilters((prev) => ({ ...prev, email: value }))
            }
          />
          <Select
            optionList={paymentStatusOptions}
            value={filters.paymentStatus}
            onChange={(value) =>
              setFilters((prev) => ({ ...prev, paymentStatus: value || '' }))
            }
          />
          <Select
            optionList={orderStatusOptions}
            value={filters.orderStatus}
            onChange={(value) =>
              setFilters((prev) => ({ ...prev, orderStatus: value || '' }))
            }
          />
        </div>

        <div className='mt-3 flex flex-wrap gap-2'>
          <Button
            type='primary'
            onClick={() => loadOrders(1, pageSize, filters)}
            loading={loading}
          >
            {t('搜索')}
          </Button>
          <Button
            type='tertiary'
            onClick={() => {
              const nextFilters = {
                orderNo: '',
                email: '',
                paymentStatus: '',
                orderStatus: '',
              };
              setFilters(nextFilters);
              loadOrders(1, pageSize, nextFilters);
            }}
          >
            {t('重置')}
          </Button>
        </div>

        <div className='mt-4'>
          <CardTable
            rowKey='order_id'
            columns={columns}
            dataSource={items}
            loading={loading}
            hidePagination
          />
        </div>

        {total > 0 && (
          <div className='mt-4 flex flex-col gap-3 md:flex-row md:items-center md:justify-between'>
            <Typography.Text type='secondary'>
              {t('共 {{count}} 条订单', { count: total })}
            </Typography.Text>
            <Pagination
              currentPage={page}
              pageSize={pageSize}
              total={total}
              pageSizeOpts={[10, 20, 50, 100]}
              showSizeChanger
              showQuickJumper
              onPageChange={(nextPage) => loadOrders(nextPage, pageSize, filters)}
              onPageSizeChange={(nextPageSize) =>
                loadOrders(1, nextPageSize, filters)
              }
            />
          </div>
        )}
      </Card>
    </div>
  );
};

export default AllergyOrders;
