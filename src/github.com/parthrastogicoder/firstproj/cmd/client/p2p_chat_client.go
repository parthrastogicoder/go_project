package main

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

// secretKey must be 16, 24, or 32 bytes long.
var secretKey = []byte("abcdefghijklmnop") // 16 bytes
 // Ensure both peers use the same key

// encrypt encrypts plaintext using AES-GCM and returns a base64-encoded ciphertext.
func encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(secretKey)
	if err != nil {
		return "", err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt decrypts a base64-encoded AES-GCM ciphertext.
func decrypt(ciphertextStr string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextStr)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(secretKey)
	if err != nil {
		return "", err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func main() {
	role := flag.String("role", "offer", "Set role: offer or answer")
	flag.Parse()

	// Connect to the signaling server.
	ws, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/ws", nil)
	if err != nil {
		log.Fatal("WebSocket dial error:", err)
	}
	defer ws.Close()

	// WebRTC configuration (using a public STUN server)
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}
	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		log.Fatal(err)
	}

	// Log ICE connection state changes.
	pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("ICE Connection State has changed: %s", state.String())
	})

	done := make(chan struct{})

	// Common function to set up the data channel.
	setupDataChannel := func(dc *webrtc.DataChannel) {
		dc.OnOpen(func() {
			log.Println("Data channel is open - you can start chatting.")
			// Start reading from stdin.
			go func() {
				scanner := bufio.NewScanner(os.Stdin)
				for scanner.Scan() {
					plainText := scanner.Text()
					log.Printf("Sending message (raw): %s", plainText)
					encrypted, err := encrypt(plainText)
					if err != nil {
						log.Println("Encryption error:", err)
						continue
					}
					if err := dc.SendText(encrypted); err != nil {
						log.Println("Send error:", err)
					}
				}
			}()
			close(done)
		})
		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			log.Printf("Received raw data: %s", msg.Data)
			decrypted, err := decrypt(string(msg.Data))
			if err != nil {
				log.Println("Decryption error:", err)
				return
			}
			fmt.Printf("Peer: %s\n", decrypted)
		})
	}

	if *role == "offer" {
		// Offerer: create a data channel.
		dataChannel, err := pc.CreateDataChannel("chat", nil)
		if err != nil {
			log.Fatal(err)
		}
		setupDataChannel(dataChannel)

		offer, err := pc.CreateOffer(nil)
		if err != nil {
			log.Fatal(err)
		}
		if err = pc.SetLocalDescription(offer); err != nil {
			log.Fatal(err)
		}
		// Wait for ICE gathering to complete.
		gatherComplete := webrtc.GatheringCompletePromise(pc)
		<-gatherComplete

		// Now send the full offer including ICE candidates.
		offerJSON, err := json.Marshal(*pc.LocalDescription())
		if err != nil {
			log.Fatal(err)
		}
		if err = ws.WriteMessage(websocket.TextMessage, offerJSON); err != nil {
			log.Fatal(err)
		}

		// Listen for answer.
		go func() {
			for {
				_, message, err := ws.ReadMessage()
				if err != nil {
					log.Println("WebSocket read error:", err)
					return
				}
				var remoteDesc webrtc.SessionDescription
				if err := json.Unmarshal(message, &remoteDesc); err != nil {
					log.Println("JSON unmarshal error:", err)
					continue
				}
				if remoteDesc.Type != webrtc.SDPTypeAnswer {
					log.Printf("Unexpected SDP type: %s (expected answer)", remoteDesc.Type)
					continue
				}
				err = pc.SetRemoteDescription(remoteDesc)
				if err != nil {
					log.Println("Error setting remote description:", err)
				} else {
					log.Println("Remote description set (offer->answer complete)")
				}
				break
			}
		}()

	} else if *role == "answer" {
		// Answerer: wait for a data channel.
		pc.OnDataChannel(func(dc *webrtc.DataChannel) {
			log.Println("Data channel received from remote.")
			setupDataChannel(dc)
		})

		// Listen for offer.
		go func() {
			for {
				_, message, err := ws.ReadMessage()
				if err != nil {
					log.Println("WebSocket read error:", err)
					return
				}
				var remoteDesc webrtc.SessionDescription
				if err := json.Unmarshal(message, &remoteDesc); err != nil {
					log.Println("JSON unmarshal error:", err)
					continue
				}
				if remoteDesc.Type != webrtc.SDPTypeOffer {
					log.Printf("Unexpected SDP type: %s (expected offer)", remoteDesc.Type)
					continue
				}
				if err := pc.SetRemoteDescription(remoteDesc); err != nil {
					log.Println("Error setting remote description:", err)
					return
				}
				log.Println("Remote offer set.")
				answer, err := pc.CreateAnswer(nil)
				if err != nil {
					log.Println("CreateAnswer error:", err)
					return
				}
				if err := pc.SetLocalDescription(answer); err != nil {
					log.Println("SetLocalDescription error:", err)
					return
				}
				// Wait for ICE gathering to complete.
				gatherComplete := webrtc.GatheringCompletePromise(pc)
				<-gatherComplete

				answerJSON, err := json.Marshal(*pc.LocalDescription())
				if err != nil {
					log.Println("JSON marshal error:", err)
					return
				}
				if err = ws.WriteMessage(websocket.TextMessage, answerJSON); err != nil {
					log.Println("WebSocket write error:", err)
					return
				}
				break
			}
		}()
	} else {
		log.Fatal("Invalid role; use -role=offer or -role=answer")
	}

	// Wait until the data channel is ready.
	<-done
	// Block forever.
	select {}
}
