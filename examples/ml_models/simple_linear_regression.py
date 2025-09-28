#!/usr/bin/env python3
"""
简单的线性回归模型示例
用于演示机器学习模型部署功能
"""

import numpy as np
import json
import sys
import os
from sklearn.linear_model import LinearRegression
from sklearn.model_selection import train_test_split
from sklearn.metrics import mean_squared_error, r2_score
import joblib

def generate_sample_data(n_samples=1000):
    """生成示例数据"""
    np.random.seed(42)
    X = np.random.randn(n_samples, 1) * 10
    y = 2 * X.flatten() + 3 + np.random.randn(n_samples) * 0.5
    return X, y

def train_model(X, y):
    """训练线性回归模型"""
    model = LinearRegression()
    model.fit(X, y)
    return model

def evaluate_model(model, X_test, y_test):
    """评估模型性能"""
    y_pred = model.predict(X_test)
    mse = mean_squared_error(y_test, y_pred)
    r2 = r2_score(y_test, y_pred)
    return mse, r2

def save_model(model, model_path):
    """保存模型"""
    os.makedirs(os.path.dirname(model_path), exist_ok=True)
    joblib.dump(model, model_path)
    print(f"模型已保存到: {model_path}")

def load_model(model_path):
    """加载模型"""
    if not os.path.exists(model_path):
        raise FileNotFoundError(f"模型文件不存在: {model_path}")
    return joblib.load(model_path)

def predict(model, X):
    """使用模型进行预测"""
    return model.predict(X)

def main():
    """主函数"""
    if len(sys.argv) < 2:
        print("用法: python simple_linear_regression.py <command> [args...]")
        print("命令:")
        print("  train <model_path> [n_samples] - 训练模型")
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
        
        print(f"开始训练模型，样本数量: {n_samples}")
        
        # 生成数据
        X, y = generate_sample_data(n_samples)
        X_train, X_test, y_train, y_test = train_test_split(X, y, test_size=0.2, random_state=42)
        
        # 训练模型
        model = train_model(X_train, y_train)
        
        # 评估模型
        mse, r2 = evaluate_model(model, X_test, y_test)
        
        # 保存模型
        save_model(model, model_path)
        
        # 输出结果
        result = {
            "status": "success",
            "model_path": model_path,
            "training_samples": len(X_train),
            "test_samples": len(X_test),
            "mse": float(mse),
            "r2_score": float(r2),
            "model_info": {
                "coefficient": float(model.coef_[0]),
                "intercept": float(model.intercept_)
            }
        }
        
        print(json.dumps(result, indent=2))
        
    elif command == "predict":
        if len(sys.argv) < 4:
            print("错误: 需要指定模型路径和输入数据")
            sys.exit(1)
        
        model_path = sys.argv[2]
        input_data = float(sys.argv[3])
        
        # 加载模型
        model = load_model(model_path)
        
        # 进行预测
        X_input = np.array([[input_data]])
        prediction = predict(model, X_input)
        
        result = {
            "status": "success",
            "input": input_data,
            "prediction": float(prediction[0]),
            "model_info": {
                "coefficient": float(model.coef_[0]),
                "intercept": float(model.intercept_)
            }
        }
        
        print(json.dumps(result, indent=2))
        
    elif command == "evaluate":
        if len(sys.argv) < 3:
            print("错误: 需要指定模型路径")
            sys.exit(1)
        
        model_path = sys.argv[2]
        
        # 加载模型
        model = load_model(model_path)
        
        # 生成测试数据
        X_test, y_test = generate_sample_data(200)
        
        # 评估模型
        mse, r2 = evaluate_model(model, X_test, y_test)
        
        result = {
            "status": "success",
            "model_path": model_path,
            "mse": float(mse),
            "r2_score": float(r2),
            "model_info": {
                "coefficient": float(model.coef_[0]),
                "intercept": float(model.intercept_)
            }
        }
        
        print(json.dumps(result, indent=2))
        
    else:
        print(f"未知命令: {command}")
        sys.exit(1)

if __name__ == "__main__":
    main()
