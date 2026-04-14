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

import React, { useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
  Button,
  Card,
  Collapse,
  Descriptions,
  Empty,
  Input,
  Select,
  Tag,
  TextArea,
  Timeline,
  Typography,
} from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from '../../helpers';

const manualStatusOptions = [
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

function formatMoney(cents, currency) {
  if (typeof cents !== 'number') return '-';
  return `${(cents / 100).toFixed(2)} ${currency || 'CNY'}`;
}

function formatAddress(address) {
  if (!address || typeof address !== 'object') return '-';
  return [
    address.province,
    address.city,
    address.district,
    address.address_line,
  ]
    .filter(Boolean)
    .join(' ');
}

function prettyJSON(raw) {
  if (!raw) return '-';
  try {
    return JSON.stringify(JSON.parse(raw), null, 2);
  } catch (error) {
    return raw;
  }
}

function getRequestErrorMessage(error, fallback) {
  return error?.response?.data?.message || error?.message || fallback;
}

function pickDefaultReportId(detail) {
  const draftReport = detail?.reports?.find(
    (item) => item.report_status !== 'published',
  );
  return (
    draftReport?.report_id ||
    detail?.current_report?.report_id ||
    detail?.reports?.[0]?.report_id ||
    ''
  );
}

const AllergyOrderDetail = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { id } = useParams();
  const fileInputRef = useRef(null);
  const [loading, setLoading] = useState(false);
  const [detail, setDetail] = useState(null);
  const [submittingAction, setSubmittingAction] = useState('');
  const [deliveryLogs, setDeliveryLogs] = useState([]);
  const [deliveryLogReportId, setDeliveryLogReportId] = useState('');
  const [statusForm, setStatusForm] = useState({
    order_status: '',
    remark: '',
  });
  const [kitForm, setKitForm] = useState({
    kit_code: '',
    outbound_carrier: '',
    outbound_tracking_no: '',
    outbound_shipped_at: '',
  });
  const [sentBackForm, setSentBackForm] = useState({
    return_tracking_no: '',
    sent_back_at: '',
    remark: '',
  });
  const [sampleReceivedForm, setSampleReceivedForm] = useState({
    received_at: '',
    remark: '',
  });
  const [testingForm, setTestingForm] = useState({
    started_at: '',
    remark: '',
  });
  const [reportForm, setReportForm] = useState({
    report_title: '过敏原检测报告',
    file: null,
  });
  const [reportAction, setReportAction] = useState({
    report_id: '',
    target_email: '',
  });
  const [completionForm, setCompletionForm] = useState({
    completed_at: '',
    remark: '',
  });

  const loadDetail = async () => {
    setLoading(true);
    try {
      const res = await API.get(`/api/admin/orders/${id}`);
      if (!res.data.success) {
        showError(res.data.message);
        return;
      }
      setDetail(res.data.data || null);
    } catch (error) {
      showError(getRequestErrorMessage(error, '获取订单详情失败'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadDetail();
  }, [id]);

  useEffect(() => {
    if (!detail) return;
    setStatusForm({
      order_status: detail.order_status || '',
      remark: detail.admin_remark || '',
    });
    setKitForm({
      kit_code: detail.sample_kit?.kit_code || '',
      outbound_carrier: detail.sample_kit?.outbound_carrier || '',
      outbound_tracking_no: detail.sample_kit?.outbound_tracking_no || '',
      outbound_shipped_at: detail.sample_kit?.outbound_shipped_at || '',
    });
    setSentBackForm((prev) => ({
      ...prev,
      return_tracking_no: detail.sample_kit?.return_tracking_no || '',
    }));
    setReportAction({
      report_id: pickDefaultReportId(detail),
      target_email: detail.recipient_email || '',
    });
  }, [detail]);

  const sortedReports = useMemo(() => detail?.reports || [], [detail]);

  const runJsonAction = async ({
    actionKey,
    method = 'post',
    url,
    payload,
    successMessage,
  }) => {
    setSubmittingAction(actionKey);
    try {
      const res = await API.request({
        method,
        url,
        data: payload,
      });
      if (!res.data.success) {
        showError(res.data.message);
        return false;
      }
      showSuccess(successMessage);
      await loadDetail();
      return true;
    } catch (error) {
      showError(getRequestErrorMessage(error, '操作失败'));
      return false;
    } finally {
      setSubmittingAction('');
    }
  };

  const uploadReport = async () => {
    if (!reportForm.file) {
      showError('请先选择 PDF 报告');
      return;
    }
    setSubmittingAction('upload_report');
    try {
      const formData = new FormData();
      formData.append('report_title', reportForm.report_title || '过敏原检测报告');
      formData.append('file', reportForm.file);
      const res = await API.post(`/api/admin/orders/${id}/report`, formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
      });
      if (!res.data.success) {
        showError(res.data.message);
        return;
      }
      showSuccess('报告上传成功');
      setReportForm({
        report_title: reportForm.report_title || '过敏原检测报告',
        file: null,
      });
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
      await loadDetail();
    } catch (error) {
      showError(getRequestErrorMessage(error, '报告上传失败'));
    } finally {
      setSubmittingAction('');
    }
  };

  const loadDeliveryLogs = async (reportId) => {
    if (!reportId) {
      showError('请先选择报告');
      return;
    }
    setSubmittingAction('load_logs');
    try {
      const res = await API.get(`/api/admin/reports/${reportId}/delivery-logs`);
      if (!res.data.success) {
        showError(res.data.message);
        return;
      }
      setDeliveryLogs(res.data.data || []);
      setDeliveryLogReportId(String(reportId));
    } catch (error) {
      showError(getRequestErrorMessage(error, '获取发送日志失败'));
    } finally {
      setSubmittingAction('');
    }
  };

  const orderStatusMeta = getStatusMeta(detail?.order_status);
  const paymentStatusMeta = getStatusMeta(detail?.payment_status);

  return (
    <div className='mt-[60px] px-2 pb-6'>
      <div className='mb-4 flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between'>
        <div>
          <Button
            theme='borderless'
            className='!px-0'
            onClick={() => navigate('/console/allergy-orders')}
          >
            {t('返回订单列表')}
          </Button>
          <Typography.Title heading={4} className='!mb-1'>
            {detail?.order_no || `#${id}`}
          </Typography.Title>
          <div className='flex flex-wrap gap-2'>
            <Tag color={orderStatusMeta.color} shape='circle' type='light'>
              {orderStatusMeta.label}
            </Tag>
            <Tag color={paymentStatusMeta.color} shape='circle' type='light'>
              {paymentStatusMeta.label}
            </Tag>
          </div>
        </div>

        <div className='flex flex-wrap gap-2'>
          <Button theme='outline' onClick={loadDetail} loading={loading}>
            {t('刷新')}
          </Button>
          <Button
            type='primary'
            onClick={() =>
              runJsonAction({
                actionKey: 'kit_preparing',
                method: 'patch',
                url: `/api/admin/orders/${id}/status`,
                payload: {
                  order_status: 'kit_preparing',
                  remark: '已通知仓库准备采样盒',
                },
                successMessage: '已标记为备货中',
              })
            }
            loading={submittingAction === 'kit_preparing'}
          >
            {t('标记备货中')}
          </Button>
        </div>
      </div>

      {loading && !detail ? (
        <Card className='!rounded-2xl shadow-sm border-0'>
          <div className='py-10 text-center text-semi-color-text-2'>
            {t('正在加载订单详情...')}
          </div>
        </Card>
      ) : null}

      {!loading && !detail ? (
        <Card className='!rounded-2xl shadow-sm border-0'>
          <Empty description={t('未找到订单详情')} />
        </Card>
      ) : null}

      {detail ? (
        <div className='grid grid-cols-1 gap-4 lg:grid-cols-2'>
          <Card className='!rounded-2xl shadow-sm border-0' title={t('订单信息')}>
            <Descriptions>
              <Descriptions.Item itemKey={t('服务名称')}>
                {detail.service_name || '-'}
              </Descriptions.Item>
              <Descriptions.Item itemKey={t('订单号')}>
                {detail.order_no || '-'}
              </Descriptions.Item>
              <Descriptions.Item itemKey={t('订单金额')}>
                {formatMoney(detail.service_price_cents, detail.currency)}
              </Descriptions.Item>
              <Descriptions.Item itemKey={t('支付时间')}>
                {formatDateTime(detail.paid_at)}
              </Descriptions.Item>
              <Descriptions.Item itemKey={t('报告就绪时间')}>
                {formatDateTime(detail.report_ready_at)}
              </Descriptions.Item>
              <Descriptions.Item itemKey={t('完成时间')}>
                {formatDateTime(detail.completed_at)}
              </Descriptions.Item>
              <Descriptions.Item itemKey={t('管理员备注')}>
                {detail.admin_remark || '-'}
              </Descriptions.Item>
            </Descriptions>
          </Card>

          <Card className='!rounded-2xl shadow-sm border-0' title={t('支付信息')}>
            <Descriptions>
              <Descriptions.Item itemKey={t('支付方式')}>
                {detail.payment_method || '-'}
              </Descriptions.Item>
              <Descriptions.Item itemKey={t('站内支付单号')}>
                {detail.payment_ref || '-'}
              </Descriptions.Item>
              <Descriptions.Item itemKey={t('三方支付单号')}>
                {detail.payment_provider_order_no || '-'}
              </Descriptions.Item>
              <Descriptions.Item itemKey={t('支付状态')}>
                <Tag color={paymentStatusMeta.color} type='light' shape='circle'>
                  {paymentStatusMeta.label}
                </Tag>
              </Descriptions.Item>
            </Descriptions>

            <div className='mt-4'>
              <Collapse>
                <Collapse.Panel header={t('原始支付回调')} itemKey='payment_raw'>
                  <pre className='max-h-[260px] overflow-auto rounded-lg bg-semi-color-fill-0 p-3 text-xs'>
                    {prettyJSON(detail.payment_callback_payload_json)}
                  </pre>
                </Collapse.Panel>
              </Collapse>
            </div>
          </Card>

          <Card className='!rounded-2xl shadow-sm border-0' title={t('收件与地址')}>
            <Descriptions>
              <Descriptions.Item itemKey={t('收件人')}>
                {detail.recipient_name || '-'}
              </Descriptions.Item>
              <Descriptions.Item itemKey={t('手机号')}>
                {detail.recipient_phone || '-'}
              </Descriptions.Item>
              <Descriptions.Item itemKey={t('邮箱')}>
                {detail.recipient_email || '-'}
              </Descriptions.Item>
              <Descriptions.Item itemKey={t('地址')}>
                {formatAddress(detail.shipping_address)}
              </Descriptions.Item>
            </Descriptions>
          </Card>

          <Card className='!rounded-2xl shadow-sm border-0' title={t('采样盒与实验室')}>
            <Descriptions>
              <Descriptions.Item itemKey={t('采样盒编码')}>
                {detail.sample_kit?.kit_code || '-'}
              </Descriptions.Item>
              <Descriptions.Item itemKey={t('采样盒状态')}>
                {detail.sample_kit?.kit_status || '-'}
              </Descriptions.Item>
              <Descriptions.Item itemKey={t('寄出物流')}>
                {detail.sample_kit?.outbound_carrier || '-'}
              </Descriptions.Item>
              <Descriptions.Item itemKey={t('寄出单号')}>
                {detail.sample_kit?.outbound_tracking_no || '-'}
              </Descriptions.Item>
              <Descriptions.Item itemKey={t('回寄单号')}>
                {detail.sample_kit?.return_tracking_no || '-'}
              </Descriptions.Item>
              <Descriptions.Item itemKey={t('寄出时间')}>
                {formatDateTime(detail.sample_kit?.outbound_shipped_at)}
              </Descriptions.Item>
              <Descriptions.Item itemKey={t('样本签收时间')}>
                {formatDateTime(detail.sample_kit?.sample_received_at)}
              </Descriptions.Item>
              <Descriptions.Item itemKey={t('实验室状态')}>
                {detail.lab_submission?.status || '-'}
              </Descriptions.Item>
              <Descriptions.Item itemKey={t('回寄追踪')}>
                {detail.lab_submission?.tracking_number || '-'}
              </Descriptions.Item>
              <Descriptions.Item itemKey={t('回寄时间')}>
                {formatDateTime(detail.lab_submission?.submitted_at)}
              </Descriptions.Item>
              <Descriptions.Item itemKey={t('开始检测时间')}>
                {formatDateTime(detail.lab_submission?.testing_started_at)}
              </Descriptions.Item>
              <Descriptions.Item itemKey={t('实验室完成时间')}>
                {formatDateTime(detail.lab_submission?.completed_at)}
              </Descriptions.Item>
            </Descriptions>
          </Card>

          <Card
            className='!rounded-2xl shadow-sm border-0 lg:col-span-2'
            title={t('引导动作')}
          >
            <div className='grid grid-cols-1 gap-4 xl:grid-cols-2'>
              <div className='rounded-xl border border-semi-color-border p-4'>
                <div className='font-semibold'>{t('1. 确认采样盒发货')}</div>
                <div className='mt-3 grid grid-cols-1 gap-3'>
                  <Input
                    placeholder={t('采样盒编码')}
                    value={kitForm.kit_code}
                    onChange={(value) =>
                      setKitForm((prev) => ({ ...prev, kit_code: value }))
                    }
                  />
                  <Input
                    placeholder={t('物流公司')}
                    value={kitForm.outbound_carrier}
                    onChange={(value) =>
                      setKitForm((prev) => ({
                        ...prev,
                        outbound_carrier: value,
                      }))
                    }
                  />
                  <Input
                    placeholder={t('物流单号')}
                    value={kitForm.outbound_tracking_no}
                    onChange={(value) =>
                      setKitForm((prev) => ({
                        ...prev,
                        outbound_tracking_no: value,
                      }))
                    }
                  />
                  <Input
                    placeholder={t('寄出时间 RFC3339，留空取当前时间')}
                    value={kitForm.outbound_shipped_at}
                    onChange={(value) =>
                      setKitForm((prev) => ({
                        ...prev,
                        outbound_shipped_at: value,
                      }))
                    }
                  />
                  <Button
                    type='primary'
                    loading={submittingAction === 'ship_kit'}
                    onClick={() =>
                      runJsonAction({
                        actionKey: 'ship_kit',
                        url: `/api/admin/orders/${id}/kit`,
                        payload: {
                          kit_code: kitForm.kit_code,
                          kit_status: 'shipped',
                          outbound_carrier: kitForm.outbound_carrier,
                          outbound_tracking_no: kitForm.outbound_tracking_no,
                          outbound_shipped_at: kitForm.outbound_shipped_at,
                        },
                        successMessage: '采样盒发货信息已更新',
                      })
                    }
                  >
                    {t('确认发货')}
                  </Button>
                </div>
              </div>

              <div className='rounded-xl border border-semi-color-border p-4'>
                <div className='font-semibold'>{t('2. 标记用户已回寄')}</div>
                <div className='mt-3 grid grid-cols-1 gap-3'>
                  <Input
                    placeholder={t('回寄单号')}
                    value={sentBackForm.return_tracking_no}
                    onChange={(value) =>
                      setSentBackForm((prev) => ({
                        ...prev,
                        return_tracking_no: value,
                      }))
                    }
                  />
                  <Input
                    placeholder={t('回寄时间 RFC3339，留空取当前时间')}
                    value={sentBackForm.sent_back_at}
                    onChange={(value) =>
                      setSentBackForm((prev) => ({
                        ...prev,
                        sent_back_at: value,
                      }))
                    }
                  />
                  <TextArea
                    placeholder={t('备注')}
                    value={sentBackForm.remark}
                    autosize={{ minRows: 2, maxRows: 4 }}
                    onChange={(value) =>
                      setSentBackForm((prev) => ({ ...prev, remark: value }))
                    }
                  />
                  <Button
                    type='primary'
                    loading={submittingAction === 'sample_sent_back'}
                    onClick={() =>
                      runJsonAction({
                        actionKey: 'sample_sent_back',
                        url: `/api/admin/orders/${id}/sample-sent-back`,
                        payload: sentBackForm,
                        successMessage: '已标记样本回寄中',
                      })
                    }
                  >
                    {t('标记已回寄')}
                  </Button>
                </div>
              </div>

              <div className='rounded-xl border border-semi-color-border p-4'>
                <div className='font-semibold'>{t('3. 标记实验室已签收')}</div>
                <div className='mt-3 grid grid-cols-1 gap-3'>
                  <Input
                    placeholder={t('签收时间 RFC3339，留空取当前时间')}
                    value={sampleReceivedForm.received_at}
                    onChange={(value) =>
                      setSampleReceivedForm((prev) => ({
                        ...prev,
                        received_at: value,
                      }))
                    }
                  />
                  <TextArea
                    placeholder={t('备注')}
                    value={sampleReceivedForm.remark}
                    autosize={{ minRows: 2, maxRows: 4 }}
                    onChange={(value) =>
                      setSampleReceivedForm((prev) => ({
                        ...prev,
                        remark: value,
                      }))
                    }
                  />
                  <Button
                    type='primary'
                    loading={submittingAction === 'sample_received'}
                    onClick={() =>
                      runJsonAction({
                        actionKey: 'sample_received',
                        url: `/api/admin/orders/${id}/sample-received`,
                        payload: sampleReceivedForm,
                        successMessage: '已标记样本签收',
                      })
                    }
                  >
                    {t('标记已签收')}
                  </Button>
                </div>
              </div>

              <div className='rounded-xl border border-semi-color-border p-4'>
                <div className='font-semibold'>{t('4. 开始检测')}</div>
                <div className='mt-3 grid grid-cols-1 gap-3'>
                  <Input
                    placeholder={t('开始检测时间 RFC3339，留空取当前时间')}
                    value={testingForm.started_at}
                    onChange={(value) =>
                      setTestingForm((prev) => ({
                        ...prev,
                        started_at: value,
                      }))
                    }
                  />
                  <TextArea
                    placeholder={t('备注')}
                    value={testingForm.remark}
                    autosize={{ minRows: 2, maxRows: 4 }}
                    onChange={(value) =>
                      setTestingForm((prev) => ({ ...prev, remark: value }))
                    }
                  />
                  <Button
                    type='primary'
                    loading={submittingAction === 'testing_started'}
                    onClick={() =>
                      runJsonAction({
                        actionKey: 'testing_started',
                        url: `/api/admin/orders/${id}/testing-started`,
                        payload: testingForm,
                        successMessage: '已进入检测阶段',
                      })
                    }
                  >
                    {t('开始检测')}
                  </Button>
                </div>
              </div>

              <div className='rounded-xl border border-semi-color-border p-4 xl:col-span-2'>
                <div className='font-semibold'>{t('5. 完成订单')}</div>
                <div className='mt-3 grid grid-cols-1 gap-3 md:grid-cols-[1fr_1fr_auto]'>
                  <Input
                    placeholder={t('完成时间 RFC3339，留空取当前时间')}
                    value={completionForm.completed_at}
                    onChange={(value) =>
                      setCompletionForm((prev) => ({
                        ...prev,
                        completed_at: value,
                      }))
                    }
                  />
                  <Input
                    placeholder={t('完成备注')}
                    value={completionForm.remark}
                    onChange={(value) =>
                      setCompletionForm((prev) => ({ ...prev, remark: value }))
                    }
                  />
                  <Button
                    type='primary'
                    loading={submittingAction === 'complete_order'}
                    onClick={() =>
                      runJsonAction({
                        actionKey: 'complete_order',
                        url: `/api/admin/orders/${id}/complete`,
                        payload: completionForm,
                        successMessage: '订单已完成',
                      })
                    }
                  >
                    {t('确认完成')}
                  </Button>
                </div>
              </div>
            </div>
          </Card>

          <Card
            className='!rounded-2xl shadow-sm border-0 lg:col-span-2'
            title={t('报告与发送')}
          >
            <div className='rounded-xl border border-semi-color-border p-4'>
              <div className='font-semibold'>{t('上传报告')}</div>
              <div className='mt-3 grid grid-cols-1 gap-3 md:grid-cols-[1fr_auto]'>
                <Input
                  placeholder={t('报告标题')}
                  value={reportForm.report_title}
                  onChange={(value) =>
                    setReportForm((prev) => ({ ...prev, report_title: value }))
                  }
                />
                <Button theme='outline' onClick={() => fileInputRef.current?.click()}>
                  {reportForm.file ? reportForm.file.name : t('选择 PDF')}
                </Button>
              </div>
              <input
                ref={fileInputRef}
                type='file'
                accept='.pdf,application/pdf'
                className='hidden'
                onChange={(event) => {
                  const file = event.target.files?.[0] || null;
                  setReportForm((prev) => ({ ...prev, file }));
                }}
              />
              <div className='mt-3'>
                <Button
                  type='primary'
                  loading={submittingAction === 'upload_report'}
                  onClick={uploadReport}
                >
                  {t('上传报告')}
                </Button>
              </div>
            </div>

            <div className='mt-4 rounded-xl border border-semi-color-border p-4'>
              <div className='font-semibold'>{t('报告列表')}</div>
              <div className='mt-3 flex flex-col gap-3'>
                {sortedReports.length === 0 ? (
                  <Empty description={t('暂无报告')} />
                ) : (
                  sortedReports.map((report) => {
                    const reportMeta = getStatusMeta(
                      report.report_status === 'published'
                        ? 'report_ready'
                        : report.report_status,
                    );
                    return (
                      <div
                        key={report.report_id}
                        className='rounded-xl bg-semi-color-fill-0 p-4'
                      >
                        <div className='flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between'>
                          <div>
                            <div className='flex flex-wrap items-center gap-2'>
                              <span className='font-semibold'>
                                {report.report_title || '-'}
                              </span>
                              <Tag color={reportMeta.color} type='light' shape='circle'>
                                {report.report_status}
                              </Tag>
                              {report.is_current ? (
                                <Tag color='green' type='solid' shape='circle'>
                                  {t('当前版本')}
                                </Tag>
                              ) : null}
                            </div>
                            <div className='mt-1 text-sm text-semi-color-text-2'>
                              {t('版本')} v{report.version} · {t('发送次数')}{' '}
                              {report.email_sent_count || 0} · {t('最近发送')}{' '}
                              {formatDateTime(report.last_email_sent_at)}
                            </div>
                          </div>
                          <div className='flex flex-wrap gap-2'>
                            {report.report_status !== 'published' ? (
                              <Button
                                type='primary'
                                theme='outline'
                                loading={
                                  submittingAction === `publish_${report.report_id}`
                                }
                                onClick={() =>
                                  runJsonAction({
                                    actionKey: `publish_${report.report_id}`,
                                    url: `/api/admin/reports/${report.report_id}/publish`,
                                    payload: {},
                                    successMessage: '报告已发布',
                                  })
                                }
                              >
                                {t('发布')}
                              </Button>
                            ) : null}
                            <Button
                              theme='outline'
                              onClick={() => window.open(report.preview_url, '_blank')}
                            >
                              {t('预览')}
                            </Button>
                            <Button
                              theme='outline'
                              onClick={() => window.open(report.download_url, '_blank')}
                            >
                              {t('下载')}
                            </Button>
                            <Button
                              theme='outline'
                              loading={submittingAction === 'load_logs'}
                              onClick={() => loadDeliveryLogs(report.report_id)}
                            >
                              {t('发送日志')}
                            </Button>
                          </div>
                        </div>
                      </div>
                    );
                  })
                )}
              </div>
            </div>

            <div className='mt-4 rounded-xl border border-semi-color-border p-4'>
              <div className='font-semibold'>{t('发送邮箱')}</div>
              <div className='mt-3 grid grid-cols-1 gap-3 lg:grid-cols-[220px_1fr_auto]'>
                <Select
                  optionList={sortedReports.map((report) => ({
                    label: `${report.report_title || '报告'} v${report.version}`,
                    value: report.report_id,
                  }))}
                  value={reportAction.report_id}
                  onChange={(value) =>
                    setReportAction((prev) => ({
                      ...prev,
                      report_id: value || '',
                    }))
                  }
                  placeholder={t('选择报告')}
                />
                <Input
                  placeholder={t('目标邮箱')}
                  value={reportAction.target_email}
                  onChange={(value) =>
                    setReportAction((prev) => ({
                      ...prev,
                      target_email: value,
                    }))
                  }
                />
                <Button
                  type='primary'
                  loading={submittingAction === 'send_email'}
                  onClick={() =>
                    runJsonAction({
                      actionKey: 'send_email',
                      url: `/api/admin/reports/${reportAction.report_id}/send-email`,
                      payload: {
                        target_email: reportAction.target_email,
                      },
                      successMessage: '报告已发送',
                    })
                  }
                  disabled={!reportAction.report_id}
                >
                  {t('发送邮件')}
                </Button>
              </div>
            </div>

            <div className='mt-4'>
              <Collapse>
                <Collapse.Panel
                  header={t('发送日志')}
                  itemKey='delivery_logs'
                >
                  <div className='mb-3 text-sm text-semi-color-text-2'>
                    {deliveryLogReportId
                      ? `${t('当前查看报告')}: #${deliveryLogReportId}`
                      : t('选择报告后可查看发送日志')}
                  </div>
                  <div className='flex flex-col gap-2'>
                    {deliveryLogs.length === 0 ? (
                      <Empty description={t('暂无发送日志')} />
                    ) : (
                      deliveryLogs.map((item, index) => (
                        <div
                          key={`${item.target}-${index}`}
                          className='rounded-xl bg-semi-color-fill-0 p-3'
                        >
                          <div className='font-medium'>{item.target}</div>
                          <div className='text-sm text-semi-color-text-2'>
                            {item.status} · {item.delivery_channel} ·{' '}
                            {formatDateTime(item.sent_at || item.created_at)}
                          </div>
                        </div>
                      ))
                    )}
                  </div>
                </Collapse.Panel>
              </Collapse>
            </div>
          </Card>

          <Card
            className='!rounded-2xl shadow-sm border-0 lg:col-span-2'
            title={t('手动状态修正')}
          >
            <div className='grid grid-cols-1 gap-3 lg:grid-cols-[220px_1fr_auto]'>
              <Select
                optionList={manualStatusOptions}
                value={statusForm.order_status}
                onChange={(value) =>
                  setStatusForm((prev) => ({
                    ...prev,
                    order_status: value || '',
                  }))
                }
                placeholder={t('选择订单状态')}
              />
              <Input
                placeholder={t('状态备注')}
                value={statusForm.remark}
                onChange={(value) =>
                  setStatusForm((prev) => ({ ...prev, remark: value }))
                }
              />
              <Button
                type='primary'
                theme='outline'
                loading={submittingAction === 'manual_status'}
                disabled={!statusForm.order_status}
                onClick={() =>
                  runJsonAction({
                    actionKey: 'manual_status',
                    method: 'patch',
                    url: `/api/admin/orders/${id}/status`,
                    payload: statusForm,
                    successMessage: '订单状态已更新',
                  })
                }
              >
                {t('更新状态')}
              </Button>
            </div>
          </Card>

          <Card
            className='!rounded-2xl shadow-sm border-0 lg:col-span-2'
            title={t('订单时间线')}
          >
            {detail.timeline?.length ? (
              <div className='max-h-[480px] overflow-y-auto pr-2'>
                <Timeline mode='left'>
                  {detail.timeline.map((item) => (
                    <Timeline.Item
                      key={`${item.event_type}-${item.occurred_at}`}
                      time={formatDateTime(item.occurred_at)}
                      extra={
                        <Tag
                          color={item.visible_to_user ? 'green' : 'grey'}
                          shape='circle'
                          type='light'
                        >
                          {item.visible_to_user ? t('会员可见') : t('仅后台')}
                        </Tag>
                      }
                    >
                      <div className='mb-1 font-semibold'>{item.title}</div>
                      <div className='text-sm text-semi-color-text-2'>
                        {item.description || '-'}
                      </div>
                      {item.event_payload_json ? (
                        <pre className='mt-2 overflow-auto rounded-lg bg-semi-color-fill-0 p-2 text-xs'>
                          {prettyJSON(item.event_payload_json)}
                        </pre>
                      ) : null}
                    </Timeline.Item>
                  ))}
                </Timeline>
              </div>
            ) : (
              <Empty description={t('暂无时间线记录')} />
            )}
          </Card>
        </div>
      ) : null}
    </div>
  );
};

export default AllergyOrderDetail;
