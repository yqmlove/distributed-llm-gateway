import { useState } from "react";

export default function App() {
  const [messages, setMessages] = useState([]);
  const [input, setInput] = useState("");
  const [loading, setLoading] = useState(false);

  const sendMessage = async () => {
    if (!input.trim()) return;

    // Add the user message to the list immediately
    setMessages((prev) => [...prev, { role: "user", content: input }]);
    setInput("");
    setLoading(true);

    // Add an empty AI message that will be filled in token by token
    setMessages((prev) => [...prev, { role: "ai", content: "" }]);

    try {
      const res = await fetch("http://localhost:8080/chat", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ message: input }),
      });

      // Get a reader to consume the SSE stream
      const reader = res.body.getReader();
      const decoder = new TextDecoder();

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        // Decode the raw bytes into a string
        const text = decoder.decode(value);
        const lines = text.split("\n");

        for (const line of lines) {
          if (!line.startsWith("data: ")) continue;

          const token = line.replace("data: ", "");
          if (token === "[DONE]") break;

          // Append the token to the last AI message in the list
          setMessages((prev) => {
            const updated = [...prev];
            updated[updated.length - 1] = {
              role: "ai",
              content: updated[updated.length - 1].content + token,
            };
            return updated;
          });
        }
      }
    } catch (err) {
      setMessages((prev) => [
        ...prev,
        { role: "ai", content: "Error: could not connect to server." },
      ]);
    } finally {
      setLoading(false);
    }
  };

  const handleKeyDown = (e) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      sendMessage();
    }
  };

  return (
    <div className="flex flex-col h-screen bg-gray-900 text-white">

      {/* Header */}
      <div className="p-4 border-b border-gray-700 text-center font-bold text-lg">
        LLM Gateway Chat
      </div>

      {/* Message list */}
      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {messages.map((msg, i) => (
          <div
            key={i}
            className={`flex ${msg.role === "user" ? "justify-end" : "justify-start"}`}
          >
            <div
              className={`max-w-xl px-4 py-2 rounded-2xl text-sm ${
                msg.role === "user"
                  ? "bg-blue-600 text-white"
                  : "bg-gray-700 text-gray-100"
              }`}
            >
              {msg.content}
            </div>
          </div>
        ))}
        {loading && (
          <div className="flex justify-start">
            <div className="bg-gray-700 px-4 py-2 rounded-2xl text-sm text-gray-400">
              Thinking...
            </div>
          </div>
        )}
      </div>

      {/* Input box */}
      <div className="p-4 border-t border-gray-700 flex gap-2">
        <input
          className="flex-1 bg-gray-800 rounded-xl px-4 py-2 text-sm outline-none"
          placeholder="Type a message..."
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={handleKeyDown}
        />
        <button
          onClick={sendMessage}
          disabled={loading}
          className="bg-blue-600 hover:bg-blue-700 px-4 py-2 rounded-xl text-sm disabled:opacity-50"
        >
          Send
        </button>
      </div>

    </div>
  );
}
