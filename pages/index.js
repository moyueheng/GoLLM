import { useState } from 'react';

export default function Home() {
    const [question, setQuestion] = useState('');
    const [answer, setAnswer] = useState('');
    const [newAnswer, setNewAnswer] = useState('');

    const handleAsk = async () => {
        const res = await fetch('http://localhost:8080/api/get-answer', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ question }),
        });
        const data = await res.json();
        setAnswer(data.answer);
    };

    const handleSave = async () => {
        await fetch('http://localhost:8080/api/save-answer', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ question, answer: newAnswer }),
        });
        setAnswer(newAnswer);
        setNewAnswer('');
    };

    return (
        <div style={{ padding: '20px' }}>
            <h1>问答系统</h1>
            <input
                type="text"
                value={question}
                onChange={(e) => setQuestion(e.target.value)}
                placeholder="请输入你的问题"
                style={{ width: '300px', padding: '8px' }}
            />
            <button onClick={handleAsk} style={{ marginLeft: '10px', padding: '8px 16px' }}>
                提问
            </button>
            <div style={{ marginTop: '20px' }}>
                <h2>回答：</h2>
                <p>{answer}</p>
                {!qaStore[question] && answer.startsWith('抱歉') && (
                    <div>
                        <input
                            type="text"
                            value={newAnswer}
                            onChange={(e) => setNewAnswer(e.target.value)}
                            placeholder="请输入答案"
                            style={{ width: '300px', padding: '8px' }}
                        />
                        <button onClick={handleSave} style={{ marginLeft: '10px', padding: '8px 16px' }}>
                            保存答案
                        </button>
                    </div>
                )}
            </div>
        </div>
    );
}
