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

import React, { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Button,
  Card,
  Input,
  InputNumber,
  Modal,
  Pagination,
  Select,
  Space,
  Tag,
  TextArea,
  Typography,
} from '@douyinfe/semi-ui';
import CardTable from '../../components/common/ui/CardTable';
import { API, showError, showSuccess } from '../../helpers';

const statusOptions = [
  { label: '草稿', value: 'draft' },
  { label: '已上架', value: 'published' },
  { label: '已下架', value: 'archived' },
];

const defaultForm = {
  service_code: '',
  title: '',
  description: '',
  image_url: '',
  cta_text: '立即购买',
  tag: '',
  price_yuan: 199,
  sort_order: 10,
  status: 'draft',
};

function formatMoney(cents, currency = 'CNY') {
  return new Intl.NumberFormat('zh-CN', {
    style: 'currency',
    currency: currency || 'CNY',
  }).format(Number(cents || 0) / 100);
}

function getStatusMeta(status) {
  const meta = {
    draft: { color: 'grey', label: '草稿' },
    published: { color: 'green', label: '已上架' },
    archived: { color: 'orange', label: '已下架' },
  };
  return meta[status] || { color: 'grey', label: status || '-' };
}

function toProductImageUrl(pathOrUrl) {
  if (!pathOrUrl) return '';
  if (pathOrUrl.startsWith('http://') || pathOrUrl.startsWith('https://')) {
    return pathOrUrl;
  }
  const baseURL = API.defaults.baseURL || '';
  if (!baseURL) return pathOrUrl;
  return `${baseURL}${pathOrUrl.startsWith('/') ? '' : '/'}${pathOrUrl}`;
}

function toForm(product) {
  return {
    service_code: product?.service_code || '',
    title: product?.title || '',
    description: product?.description || '',
    image_url: product?.image_url || '',
    cta_text: product?.cta_text || '立即购买',
    tag: product?.tag || '',
    price_yuan: Number(((product?.price_cents || 0) / 100).toFixed(2)),
    sort_order: Number(product?.sort_order || 0),
    status: product?.status || 'draft',
  };
}

function toPayload(form, includeCode) {
  const payload = {
    title: form.title.trim(),
    description: form.description.trim(),
    image_url: form.image_url.trim(),
    cta_text: form.cta_text.trim(),
    tag: form.tag.trim(),
    price_cents: Math.round(Number(form.price_yuan || 0) * 100),
    sort_order: Number(form.sort_order || 0),
    status: form.status,
  };
  if (includeCode) {
    payload.service_code = form.service_code.trim();
  }
  return payload;
}

function getRequestErrorMessage(error, fallback) {
  return error?.response?.data?.message || error?.message || fallback;
}

const AllergyServiceProducts = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [items, setItems] = useState([]);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingProduct, setEditingProduct] = useState(null);
  const [form, setForm] = useState(defaultForm);
  const [imageUploading, setImageUploading] = useState(false);
  const imageInputRef = useRef(null);

  const loadProducts = async (nextPage = page, nextPageSize = pageSize) => {
    setLoading(true);
    try {
      const res = await API.get('/api/admin/service-products', {
        params: {
          p: nextPage,
          page_size: nextPageSize,
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
      showError(getRequestErrorMessage(error, '获取检测项目失败'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadProducts(1, pageSize);
  }, []);

  const openCreateModal = () => {
    setEditingProduct(null);
    setForm(defaultForm);
    setModalVisible(true);
  };

  const openEditModal = async (record) => {
    setSaving(true);
    try {
      const res = await API.get(`/api/admin/service-products/${record.id}`);
      if (!res.data.success) {
        showError(res.data.message);
        return;
      }
      setEditingProduct(res.data.data);
      setForm(toForm(res.data.data));
      setModalVisible(true);
    } catch (error) {
      showError(getRequestErrorMessage(error, '获取检测项目详情失败'));
    } finally {
      setSaving(false);
    }
  };

  const closeModal = () => {
    setModalVisible(false);
    setEditingProduct(null);
    setForm(defaultForm);
    if (imageInputRef.current) {
      imageInputRef.current.value = '';
    }
  };

  const uploadProductImage = async (file) => {
    if (!file) return;
    setImageUploading(true);
    try {
      const formData = new FormData();
      formData.append('file', file);
      const res = await API.post('/api/admin/service-products/upload-image', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
      });
      if (!res.data.success) {
        showError(res.data.message);
        return;
      }
      const imageUrl = res.data.data?.image_url || '';
      setForm((prev) => ({ ...prev, image_url: imageUrl }));
      showSuccess('图片上传成功');
    } catch (error) {
      showError(getRequestErrorMessage(error, '图片上传失败'));
    } finally {
      if (imageInputRef.current) {
        imageInputRef.current.value = '';
      }
      setImageUploading(false);
    }
  };

  const handleImageFileChange = async (event) => {
    const file = event.target.files?.[0];
    await uploadProductImage(file);
  };

  const submitProduct = async () => {
    setSaving(true);
    try {
      const isEdit = Boolean(editingProduct?.id);
      const payload = toPayload(form, !isEdit);
      const res = isEdit
        ? await API.patch(
            `/api/admin/service-products/${editingProduct.id}`,
            payload,
          )
        : await API.post('/api/admin/service-products', payload);
      if (!res.data.success) {
        showError(res.data.message);
        return;
      }
      showSuccess(isEdit ? '检测项目已更新' : '检测项目已创建');
      closeModal();
      await loadProducts(page, pageSize);
    } catch (error) {
      showError(getRequestErrorMessage(error, '保存检测项目失败'));
    } finally {
      setSaving(false);
    }
  };

  const updateStatus = async (record, nextStatus) => {
    const actionLabel = nextStatus === 'published' ? '上架' : '下架';
    Modal.confirm({
      title: `${actionLabel}检测项目`,
      content: `确认${actionLabel}「${record.title}」？`,
      onOk: async () => {
        try {
          const res = await API.post(
            `/api/admin/service-products/${record.id}/${nextStatus === 'published' ? 'publish' : 'archive'}`,
          );
          if (!res.data.success) {
            showError(res.data.message);
            return;
          }
          showSuccess(`检测项目已${actionLabel}`);
          await loadProducts(page, pageSize);
        } catch (error) {
          showError(
            getRequestErrorMessage(error, `${actionLabel}检测项目失败`),
          );
        }
      },
    });
  };

  const columns = [
    {
      title: t('检测项目'),
      dataIndex: 'title',
      key: 'title',
      render: (value, record) => (
        <div className='flex flex-col'>
          <span className='font-semibold'>{value || '-'}</span>
          <span className='text-xs text-semi-color-text-2'>
            {record.service_code}
          </span>
        </div>
      ),
    },
    {
      title: t('价格'),
      dataIndex: 'price_cents',
      key: 'price_cents',
      render: (value, record) => formatMoney(value, record.currency),
    },
    {
      title: t('状态'),
      dataIndex: 'status',
      key: 'status',
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
      title: t('排序'),
      dataIndex: 'sort_order',
      key: 'sort_order',
    },
    {
      title: t('操作'),
      key: 'action',
      render: (_, record) => (
        <Space>
          <Button
            size='small'
            theme='outline'
            onClick={() => openEditModal(record)}
          >
            {t('编辑')}
          </Button>
          {record.status === 'published' ? (
            <Button
              size='small'
              type='warning'
              theme='outline'
              onClick={() => updateStatus(record, 'archived')}
            >
              {t('下架')}
            </Button>
          ) : (
            <Button
              size='small'
              type='primary'
              theme='outline'
              onClick={() => updateStatus(record, 'published')}
            >
              {t('上架')}
            </Button>
          )}
        </Space>
      ),
    },
  ];

  return (
    <div className='mt-[60px] px-2'>
      <Card>
        <div className='mb-4 flex flex-col gap-3 md:flex-row md:items-center md:justify-between'>
          <div>
            <Typography.Title heading={4} style={{ margin: 0 }}>
              {t('检测项目')}
            </Typography.Title>
            <Typography.Text type='secondary'>
              {t('管理前台可售检测项目、价格和商品详情')}
            </Typography.Text>
          </div>
          <Button type='primary' onClick={openCreateModal}>
            {t('新增检测项目')}
          </Button>
        </div>

        <CardTable
          columns={columns}
          dataSource={items}
          rowKey='id'
          loading={loading}
          hidePagination
        />

        <div className='mt-4 flex justify-end'>
          <Pagination
            currentPage={page}
            pageSize={pageSize}
            total={total}
            showSizeChanger
            onPageChange={(nextPage) => loadProducts(nextPage, pageSize)}
            onPageSizeChange={(nextPageSize) => loadProducts(1, nextPageSize)}
          />
        </div>
      </Card>

      <Modal
        title={editingProduct ? '编辑检测项目' : '新增检测项目'}
        visible={modalVisible}
        onCancel={closeModal}
        onOk={submitProduct}
        confirmLoading={saving}
        okText={t('保存')}
        cancelText={t('取消')}
        width={720}
      >
        <div className='grid gap-4'>
          <label className='grid gap-2'>
            <span className='text-sm font-medium'>{t('服务编码')}</span>
            <Input
              value={form.service_code}
              disabled={Boolean(editingProduct)}
              placeholder='allergy-test-basic'
              onChange={(value) => setForm({ ...form, service_code: value })}
            />
          </label>
          <label className='grid gap-2'>
            <span className='text-sm font-medium'>{t('项目标题')}</span>
            <Input
              value={form.title}
              onChange={(value) => setForm({ ...form, title: value })}
            />
          </label>
          <label className='grid gap-2'>
            <span className='text-sm font-medium'>{t('项目详情')}</span>
            <TextArea
              autosize={{ minRows: 3, maxRows: 6 }}
              value={form.description}
              onChange={(value) => setForm({ ...form, description: value })}
            />
          </label>
          <label className='grid gap-2'>
            <span className='text-sm font-medium'>{t('图片 URL')}</span>
            <div className='flex flex-col gap-3 md:flex-row'>
              <Input
                value={form.image_url}
                onChange={(value) => setForm({ ...form, image_url: value })}
              />
              <Button
                theme='outline'
                loading={imageUploading}
                onClick={() => imageInputRef.current?.click()}
              >
                {t('上传本地图片')}
              </Button>
            </div>
            <input
              ref={imageInputRef}
              type='file'
              accept='image/*'
              className='hidden'
              onChange={handleImageFileChange}
            />
            {form.image_url ? (
              <div className='overflow-hidden rounded border border-semi-color-border bg-semi-color-bg-0'>
                <img
                  src={toProductImageUrl(form.image_url)}
                  alt={form.title || '检测项目图片预览'}
                  className='h-40 w-full object-cover'
                />
              </div>
            ) : null}
          </label>
          <div className='grid gap-4 md:grid-cols-2'>
            <label className='grid gap-2'>
              <span className='text-sm font-medium'>{t('CTA 文案')}</span>
              <Input
                value={form.cta_text}
                onChange={(value) => setForm({ ...form, cta_text: value })}
              />
            </label>
            <label className='grid gap-2'>
              <span className='text-sm font-medium'>{t('标签')}</span>
              <Input
                value={form.tag}
                onChange={(value) => setForm({ ...form, tag: value })}
              />
            </label>
          </div>
          <div className='grid gap-4 md:grid-cols-3'>
            <label className='grid gap-2'>
              <span className='text-sm font-medium'>{t('价格（元）')}</span>
              <InputNumber
                min={0.01}
                precision={2}
                value={form.price_yuan}
                onChange={(value) => setForm({ ...form, price_yuan: value })}
              />
            </label>
            <label className='grid gap-2'>
              <span className='text-sm font-medium'>{t('排序')}</span>
              <InputNumber
                value={form.sort_order}
                onChange={(value) => setForm({ ...form, sort_order: value })}
              />
            </label>
            <label className='grid gap-2'>
              <span className='text-sm font-medium'>{t('状态')}</span>
              <Select
                value={form.status}
                optionList={statusOptions}
                onChange={(value) => setForm({ ...form, status: value })}
              />
            </label>
          </div>
        </div>
      </Modal>
    </div>
  );
};

export default AllergyServiceProducts;
