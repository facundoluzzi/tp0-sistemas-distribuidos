import json
import os
import socket
import logging
import signal
import struct
import sys

from common.utils import Bet, store_bets
from common import utils

class Server:
    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self.client_connections = []
        self.is_running = True
        self.max_connections = int(os.environ.get("MAX_CONNECTIONS", 5))
        
        signal.signal(signal.SIGTERM, self.graceful_shutdown)
        signal.signal(signal.SIGINT, self.graceful_shutdown)
        
    def run(self):
        """
        Dummy Server loop

        Server that accept a new connections and establishes a
        communication with a client. After client with communucation
        finishes, servers starts to accept new connections again
        """
        while self.is_running:
            client_sock = self.__accept_new_connection()
            
            if len(self.client_connections) >= self.max_connections:
                logging.info(f'max clients connected {self.max_connections}, rejecting new connection')
                client_sock.sendall(b"ERROR: Maximum number of agencies reached\n")
                client_sock.close()
                continue 
            
            self.client_connections.append(client_sock)
            self.__handle_client_connection(client_sock)
            
    def __handle_client_connection(self, client_sock):
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
        try:
            length_data = client_sock.recv(2)
            if not length_data:
                return  

            msg_length = struct.unpack("!H", length_data)[0]
            msg = client_sock.recv(msg_length).decode().strip()

            if not msg:
                return  
            
            lines = msg.split("\n")
            msg_type = lines[0]
            
            addr = client_sock.getpeername()
            logging.info(f'action: receive_message | result: success | ip: {addr[0]} | msg: {msg}')

            if msg_type == "bet":
                parts = lines[1].split("|")
                if len(parts) != 6:
                    logging.warning(f"action: parse_bet | result: fail | reason: invalid_format | data: {line}")
                    return  

                bet = Bet(*parts)
                
                store_bets([bet])
                
                logging.info(f'action: apuesta_almacenada | result: success | dni: {bet.document} | numero: {bet.number}')
                
                ack_response = utils.ACK_MESSAGE.format(bet.agency)

                client_sock.sendall(ack_response.encode('utf-8'))
        except OSError as e:
            logging.error("action: receive_message | result: fail | error: {e}")
        finally:
            client_sock.close()
            self.client_connections.remove(client_sock)
            
    def __accept_new_connection(self):
        """
        Accept new connections

        Function blocks until a connection to a client is made.
        Then connection created is printed and returned
        """
        # Connection arrived
        logging.info('action: accept_connections | result: in_progress')
        c, addr = self._server_socket.accept()
        logging.info(f'action: accept_connections | result: success | ip: {addr[0]}')
        return c
        
    def graceful_shutdown(self, signum, frame):
        self.is_running = False
        
        logging.info(f"stopping server due to received signal: {signum}")
        self._server_socket.close()
        logging.info("server socket was closed")
        
        logging.info(f"closing {len(self.client_connections)} client connections")

        for conn in self.client_connections:
            conn.close()
            
        logging.info("client connections were closed successfully")
        
        sys.exit(0)