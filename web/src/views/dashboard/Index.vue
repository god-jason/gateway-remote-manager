<template>
  <div>
    <el-row :gutter="16">
      <el-col :span="8">
        <el-card>
          <div class="stat">
            <div class="label">设备总数</div>
            <div class="value">{{ summary.total }}</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card>
          <div class="stat">
            <div class="label">在线</div>
            <div class="value success">{{ summary.online }}</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card>
          <div class="stat">
            <div class="label">离线</div>
            <div class="value danger">{{ summary.offline }}</div>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { dashboardSummary } from '@/api/device'

const summary = ref({ total: 0, online: 0, offline: 0 })

async function load() {
  const res = await dashboardSummary()
  summary.value = res.data
}

onMounted(load)
</script>

<style scoped>
.stat {
  text-align: center;
  padding: 8px 0;
}
.label {
  color: #909399;
  font-size: 14px;
}
.value {
  font-size: 32px;
  font-weight: 700;
  margin-top: 8px;
}
.success {
  color: #67c23a;
}
.danger {
  color: #f56c6c;
}
</style>