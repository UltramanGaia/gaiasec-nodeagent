FROM alpine:3.7
ADD kubectl /bin/kubectl
ADD agent/ /agent/
ADD sync.sh /
RUN chmod +x /sync.sh
CMD ["/sync.sh"]