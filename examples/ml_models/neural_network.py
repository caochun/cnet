#!/usr/bin/env python3
"""
简单的神经网络模型示例
用于演示深度学习模型部署功能
"""

import numpy as np
import json
import sys
import os
import tensorflow as tf
from tensorflow import keras
from tensorflow.keras import layers
from sklearn.model_selection import train_test_split
from sklearn.preprocessing import StandardScaler
from sklearn.metrics import mean_squared_error, r2_score

def generate_nonlinear_data(n_samples=1000):
    """生成非线性示例数据"""
    np.random.seed(42)
    X = np.random.randn(n_samples, 2) * 5
    # 非线性关系: y = x1^2 + x2^2 + 0.5*x1*x2 + noise
    y = X[:, 0]**2 + X[:, 1]**2 + 0.5 * X[:, 0] * X[:, 1] + np.random.randn(n_samples) * 0.5
    return X, y

def create_model(input_dim=2):
    """创建神经网络模型"""
    model = keras.Sequential([
        layers.Dense(64, activation='relu', input_shape=(input_dim,)),
        layers.Dropout(0.2),
        layers.Dense(32, activation='relu'),
        layers.Dropout(0.2),
        layers.Dense(16, activation='relu'),
        layers.Dense(1)
    ])
    
    model.compile(
        optimizer='adam',
        loss='mse',
        metrics=['mae']
    )
    
    return model

def train_model(model, X_train, y_train, X_val, y_val, epochs=100, batch_size=32):
    """训练模型"""
    history = model.fit(
        X_train, y_train,
        validation_data=(X_val, y_val),
        epochs=epochs,
        batch_size=batch_size,
        verbose=0
    )
    return history

def evaluate_model(model, X_test, y_test):
    """评估模型性能"""
    y_pred = model.predict(X_test, verbose=0)
    mse = mean_squared_error(y_test, y_pred)
    r2 = r2_score(y_test, y_pred)
    return mse, r2

def save_model(model, model_path):
    """保存模型"""
    os.makedirs(os.path.dirname(model_path), exist_ok=True)
    model.save(model_path)
    print(f"模型已保存到: {model_path}")

def load_model(model_path):
    """加载模型"""
    if not os.path.exists(model_path):
        raise FileNotFoundError(f"模型文件不存在: {model_path}")
    return keras.models.load_model(model_path)

def predict(model, X):
    """使用模型进行预测"""
    return model.predict(X, verbose=0)

def main():
    """主函数"""
    if len(sys.argv) < 2:
        print("用法: python neural_network.py <command> [args...]")
        print("命令:")
        print("  train <model_path> [n_samples] [epochs] - 训练模型")
        print("  predict <model_path> <input_data> - 使用模型预测")
        print("  evaluate <model_path> - 评估模型")
        sys.exit(1)
    
    command = sys.argv[1]
    
    if command == "train":
        if len(sys.argv) < 3:
            print("错误: 需要指定模型保存路径")
            sys.exit(1)
        
        model_path = sys.argv[2]
        n_samples = int(sys.argv[3]) if len(sys.argv) > 3 else 1000
        epochs = int(sys.argv[4]) if len(sys.argv) > 4 else 100
        
        print(f"开始训练神经网络模型，样本数量: {n_samples}, 训练轮数: {epochs}")
        
        # 生成数据
        X, y = generate_nonlinear_data(n_samples)
        
        # 数据标准化
        scaler = StandardScaler()
        X_scaled = scaler.fit_transform(X)
        
        # 分割数据
        X_train, X_test, y_train, y_test = train_test_split(
            X_scaled, y, test_size=0.2, random_state=42
        )
        X_train, X_val, y_train, y_val = train_test_split(
            X_train, y_train, test_size=0.2, random_state=42
        )
        
        # 创建和训练模型
        model = create_model(input_dim=X.shape[1])
        history = train_model(model, X_train, y_train, X_val, y_val, epochs)
        
        # 评估模型
        mse, r2 = evaluate_model(model, X_test, y_test)
        
        # 保存模型和标准化器
        save_model(model, model_path)
        scaler_path = model_path.replace('.h5', '_scaler.joblib')
        import joblib
        joblib.dump(scaler, scaler_path)
        
        # 输出结果
        result = {
            "status": "success",
            "model_path": model_path,
            "scaler_path": scaler_path,
            "training_samples": len(X_train),
            "validation_samples": len(X_val),
            "test_samples": len(X_test),
            "mse": float(mse),
            "r2_score": float(r2),
            "epochs": epochs,
            "final_loss": float(history.history['loss'][-1]),
            "final_val_loss": float(history.history['val_loss'][-1])
        }
        
        print(json.dumps(result, indent=2))
        
    elif command == "predict":
        if len(sys.argv) < 4:
            print("错误: 需要指定模型路径和输入数据")
            sys.exit(1)
        
        model_path = sys.argv[2]
        input_data = [float(x) for x in sys.argv[3].split(',')]
        
        if len(input_data) != 2:
            print("错误: 输入数据必须是两个数字，用逗号分隔")
            sys.exit(1)
        
        # 加载模型和标准化器
        model = load_model(model_path)
        scaler_path = model_path.replace('.h5', '_scaler.joblib')
        import joblib
        scaler = joblib.load(scaler_path)
        
        # 进行预测
        X_input = np.array([input_data])
        X_input_scaled = scaler.transform(X_input)
        prediction = predict(model, X_input_scaled)
        
        result = {
            "status": "success",
            "input": input_data,
            "prediction": float(prediction[0][0]),
            "model_type": "neural_network"
        }
        
        print(json.dumps(result, indent=2))
        
    elif command == "evaluate":
        if len(sys.argv) < 3:
            print("错误: 需要指定模型路径")
            sys.exit(1)
        
        model_path = sys.argv[2]
        
        # 加载模型和标准化器
        model = load_model(model_path)
        scaler_path = model_path.replace('.h5', '_scaler.joblib')
        import joblib
        scaler = joblib.load(scaler_path)
        
        # 生成测试数据
        X_test, y_test = generate_nonlinear_data(200)
        X_test_scaled = scaler.transform(X_test)
        
        # 评估模型
        mse, r2 = evaluate_model(model, X_test_scaled, y_test)
        
        result = {
            "status": "success",
            "model_path": model_path,
            "mse": float(mse),
            "r2_score": float(r2),
            "model_type": "neural_network"
        }
        
        print(json.dumps(result, indent=2))
        
    else:
        print(f"未知命令: {command}")
        sys.exit(1)

if __name__ == "__main__":
    main()
