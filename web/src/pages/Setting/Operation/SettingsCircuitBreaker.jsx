import React, { useEffect, useState, useRef } from 'react';
import {
  Button,
  Col,
  Form,
  Row,
  Spin,
  Tag,
  Space,
  Typography,
  Divider,
  Empty,
} from '@douyinfe/semi-ui';
import {
  compareObjects,
  API,
  showError,
  showSuccess,
  showWarning,
} from '../../../helpers';
import { useTranslation } from 'react-i18next';

const { Text } = Typography;

export default function SettingsCircuitBreaker(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [statusLoading, setStatusLoading] = useState(false);
  const [circuitStatuses, setCircuitStatuses] = useState([]);
  const [inputs, setInputs] = useState({
    CircuitBreakerEnabled: false,
    CircuitBreakerWindowSize: '',
    CircuitBreakerFailureThreshold: '',
    CircuitBreakerInitialCooldownSeconds: '',
    CircuitBreakerMaxCooldownSeconds: '',
  });
  const refForm = useRef();
  const [inputsRow, setInputsRow] = useState(inputs);

  function handleFieldChange(fieldName) {
    return (value) => {
      setInputs((prev) => ({ ...prev, [fieldName]: value }));
    };
  }

  async function fetchCircuitStatus() {
    setStatusLoading(true);
    try {
      const res = await API.get('/api/option/circuit_breaker');
      if (res.data.success) {
        setCircuitStatuses(res.data.data || []);
      } else {
        showError(res.data.message);
      }
    } catch {
      showError(t('获取熔断状态失败'));
    } finally {
      setStatusLoading(false);
    }
  }

  async function handleReset(channelId) {
    try {
      const res = await API.delete(`/api/option/circuit_breaker/${channelId}`);
      if (res.data.success) {
        showSuccess(t('已手动解除熔断'));
        fetchCircuitStatus();
      } else {
        showError(res.data.message);
      }
    } catch {
      showError(t('操作失败'));
    }
  }

  function onSubmit() {
    const updateArray = compareObjects(inputs, inputsRow);
    if (!updateArray.length) return showWarning(t('你似乎并没有修改什么'));
    const requestQueue = updateArray.map((item) => {
      const value =
        typeof inputs[item.key] === 'boolean'
          ? String(inputs[item.key])
          : inputs[item.key];
      return API.put('/api/option/', { key: item.key, value });
    });
    setLoading(true);
    Promise.all(requestQueue)
      .then((res) => {
        if (res.includes(undefined)) return showError(t('部分保存失败，请重试'));
        showSuccess(t('保存成功'));
        props.refresh();
      })
      .catch(() => showError(t('保存失败，请重试')))
      .finally(() => setLoading(false));
  }

  useEffect(() => {
    const currentInputs = {};
    for (let key in props.options) {
      if (Object.keys(inputs).includes(key)) {
        currentInputs[key] = props.options[key];
      }
    }
    setInputs(currentInputs);
    setInputsRow(structuredClone(currentInputs));
    if (refForm.current) refForm.current.setValues(currentInputs);
  }, [props.options]);

  useEffect(() => {
    fetchCircuitStatus();
  }, []);

  const openStatuses = circuitStatuses.filter((s) => s.is_open);

  function formatSeconds(s) {
    if (s >= 3600) return `${Math.floor(s / 3600)} 小时 ${Math.floor((s % 3600) / 60)} 分钟`;
    if (s >= 60) return `${Math.floor(s / 60)} 分钟 ${s % 60} 秒`;
    return `${s} 秒`;
  }

  return (
    <Spin spinning={loading}>
      <Form
        values={inputs}
        getFormApi={(api) => (refForm.current = api)}
        style={{ marginTop: 10 }}
      >
        <Row gutter={[16, 16]}>
          <Col xs={24} sm={12} md={6} lg={6} xl={6}>
            <Form.Switch
              field="CircuitBreakerEnabled"
              label={t('启用渠道熔断')}
              checkedText={t('开')}
              uncheckedText={t('关')}
              onChange={handleFieldChange('CircuitBreakerEnabled')}
            />
          </Col>
          <Col xs={24} sm={12} md={6} lg={6} xl={6}>
            <Form.Input
              field="CircuitBreakerWindowSize"
              label={t('滑动窗口大小（请求次数）')}
              placeholder="10"
              onChange={handleFieldChange('CircuitBreakerWindowSize')}
              showClear
            />
          </Col>
          <Col xs={24} sm={12} md={6} lg={6} xl={6}>
            <Form.Input
              field="CircuitBreakerFailureThreshold"
              label={t('触发熔断失败次数')}
              placeholder="3"
              onChange={handleFieldChange('CircuitBreakerFailureThreshold')}
              showClear
            />
          </Col>
          <Col xs={24} sm={12} md={6} lg={6} xl={6}>
            <Form.Input
              field="CircuitBreakerInitialCooldownSeconds"
              label={t('初始冷却时间（秒）')}
              placeholder="300"
              onChange={handleFieldChange('CircuitBreakerInitialCooldownSeconds')}
              showClear
            />
          </Col>
          <Col xs={24} sm={12} md={6} lg={6} xl={6}>
            <Form.Input
              field="CircuitBreakerMaxCooldownSeconds"
              label={t('最大冷却时间（秒）')}
              placeholder="7200"
              onChange={handleFieldChange('CircuitBreakerMaxCooldownSeconds')}
              showClear
            />
          </Col>
        </Row>

        <Row style={{ marginTop: 16 }}>
          <Col>
            <Space>
              <Button type="primary" onClick={onSubmit}>
                {t('保存熔断设置')}
              </Button>
            </Space>
          </Col>
        </Row>
      </Form>

      <Divider style={{ margin: '20px 0 12px' }}>
        {t('当前熔断中的渠道')}
        <Button
          size="small"
          theme="borderless"
          style={{ marginLeft: 8 }}
          loading={statusLoading}
          onClick={fetchCircuitStatus}
        >
          {t('刷新')}
        </Button>
      </Divider>

      <Spin spinning={statusLoading}>
        {openStatuses.length === 0 ? (
          <Empty
            description={t('当前没有渠道处于熔断状态')}
            style={{ padding: '20px 0' }}
          />
        ) : (
          <Row gutter={[12, 12]}>
            {openStatuses.map((s) => (
              <Col key={s.channel_id} xs={24} sm={12} md={8} lg={6}>
                <div
                  style={{
                    border: '1px solid var(--semi-color-danger-light-active)',
                    borderRadius: 8,
                    padding: '12px 16px',
                    background: 'var(--semi-color-danger-light-default)',
                  }}
                >
                  <Space vertical align="start" style={{ width: '100%' }}>
                    <Space>
                      <Text strong>{t('渠道')} #{s.channel_id}</Text>
                      <Tag color="red" size="small">
                        {t('熔断中')}
                      </Tag>
                    </Space>
                    <Text type="tertiary" size="small">
                      {t('剩余冷却')}：{formatSeconds(s.cooldown_seconds)}
                    </Text>
                    <Text type="tertiary" size="small">
                      {t('累计熔断次数')}：{s.trip_count} {t('次')}（{t('下次冷却')}：
                      {formatSeconds(
                        Math.min(
                          parseInt(inputs.CircuitBreakerInitialCooldownSeconds || 300) *
                            Math.pow(2, s.trip_count),
                          parseInt(inputs.CircuitBreakerMaxCooldownSeconds || 7200),
                        ),
                      )}）
                    </Text>
                    <Button
                      size="small"
                      type="danger"
                      theme="light"
                      onClick={() => handleReset(s.channel_id)}
                    >
                      {t('手动解除熔断')}
                    </Button>
                  </Space>
                </div>
              </Col>
            ))}
          </Row>
        )}
      </Spin>
    </Spin>
  );
}
