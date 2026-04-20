<template>
  <div class="not-found-page">
    <div class="container">
      <div class="status-code">404</div>
      <div class="status-text">页面未找到</div>
      <div class="desc">您访问的资源不存在或已被移除</div>
      <div class="divider"></div>
      <router-link to="/" class="btn btn-primary">返回首页</router-link>
    </div>
  </div>
</template>

<script setup lang="ts">
</script>

<style scoped>
.not-found-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f5f7fa;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  color: #333;
}

.container {
  text-align: center;
  padding: 40px;
  max-width: 480px;
}

.status-code {
  font-size: 72px;
  font-weight: 800;
  color: #c0c4cc;
  margin-bottom: 0;
  line-height: 1;
}

.status-text {
  font-size: 18px;
  color: #909399;
  margin-bottom: 24px;
}

.desc {
  font-size: 14px;
  color: #999;
  margin-bottom: 32px;
  line-height: 1.6;
}

.divider {
  width: 40px;
  height: 2px;
  background: #e4e7ed;
  margin: 0 auto 16px;
  border-radius: 1px;
}

.btn {
  display: inline-block;
  padding: 10px 24px;
  font-size: 14px;
  color: #fff;
  text-decoration: none;
  border-radius: 6px;
  transition: background 0.2s;
}

.btn-primary {
  background: #409eff;
}

.btn-primary:hover {
  background: #337ecc;
}
</style>
